// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import http from 'node:http';
import { signIn, unique, createSpace } from './helpers.js';

const LD_PAGE = `<html><head><script type="application/ld+json">{"@type":"Recipe","name":"Linsendal","recipeYield":"4",
"recipeIngredient":["250 g rote Linsen","1 Dose Kokosmilch"],"recipeInstructions":[{"@type":"HowToStep","text":"Kochen."}]}</script></head><body></body></html>`;
const PLAIN_PAGE = '<html><body><h1>Apfelkuchen</h1><p>500 g Äpfel</p></body></html>';
const AI_ANSWER = '# Apfelkuchen\nPortionen: 12\n\n## Zutaten\n\n- 500 g Äpfel\n\n## Zubereitung\n\n1. Backen.';

/** Starts a local server; the dev login lets Kanso fetch from loopback. */
function serve(handler) {
  return new Promise((resolve) => {
    const server = http.createServer(handler).listen(0, '127.0.0.1', () => resolve(server));
  });
}

let site;
let fakeAI;
test.beforeAll(async () => {
  site = await serve((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(req.url === '/curry' ? LD_PAGE : PLAIN_PAGE);
  });
  fakeAI = await serve((req, res) => {
    req.resume();
    req.on('end', () => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ choices: [{ message: { content: req.url.endsWith('/chat/completions') ? AI_ANSWER : 'ok' } }] }));
    });
  });
});
test.afterAll(() => { site?.close(); fakeAI?.close(); });

const at = (server, path) => `http://127.0.0.1:${server.address().port}${path}`;

test('recipe from a web page: recipe data, then with the AI after an admin connects one', async ({ page }) => {
  await signIn(page, { sub: 'ines', name: 'Ines', admin: true });
  const space = unique('Importe');
  await createSpace(page, space);

  await page.goto('/recipes');
  await page.getByRole('link', { name: 'Von Webseite' }).click();
  await page.getByLabel('Adresse der Rezeptseite').fill(at(site, '/curry'));
  await page.getByLabel('Wiki-Bereich').selectOption({ label: space });
  await page.getByRole('button', { name: 'Rezept holen' }).click();
  await expect(page.getByLabel('Titel')).toHaveValue('Linsendal');
  await expect(page.locator('.recipe-parsed')).toContainText('250 · g · rote Linsen');
  await page.getByRole('button', { name: 'Rezept anlegen' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}\/edit$/);

  await page.goto('/admin/ai');
  await page.getByLabel('KI eingeschaltet').check();
  await page.getByLabel('Basis-URL').fill(at(fakeAI, '/v1'));
  await page.getByLabel('Modell').fill('test-model');
  await page.getByLabel('Token', { exact: true }).fill('sk-test-geheim');
  await page.getByRole('button', { name: 'Speichern und Verbindung testen' }).click();
  await expect(page.getByRole('status')).toHaveText('Verbindung steht: Die KI hat geantwortet.');
  await expect(page.locator('main')).not.toContainText('sk-test-geheim');

  await page.goto('/recipes/import');
  await expect(page.locator('.ai-mark')).toHaveText('KI · im eigenen Netz: 127.0.0.1');
  await page.getByLabel('Adresse der Rezeptseite').fill(at(site, '/apfel'));
  await page.getByLabel('Wiki-Bereich').selectOption({ label: space });
  await page.getByRole('button', { name: 'Rezept holen' }).click();
  await expect(page.getByLabel('Titel')).toHaveValue('Apfelkuchen');
  await expect(page.locator('main')).toContainText('mit KI aus dem Text der Seite');

  await page.goto('/admin/ai');
  await page.getByLabel('KI eingeschaltet').uncheck();
  await page.getByRole('button', { name: 'Speichern', exact: true }).click();
});
