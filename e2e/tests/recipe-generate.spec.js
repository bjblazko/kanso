// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import http from 'node:http';
import { signIn, unique, createSpace, useAsCookbook } from './helpers.js';

/** A fake AI that makes up a different dish on every call. */
let calls = 0;
let fakeAI;
test.beforeAll(async () => {
  fakeAI = await new Promise((resolve) => {
    const server = http.createServer((req, res) => {
      req.resume();
      req.on('end', () => {
        calls++;
        const content = req.url.endsWith('/chat/completions')
          ? `# Pfanne Nummer ${calls}\nPortionen: 4\nZeit: 30\n\nDeftig.\n\n## Zutaten\n\n- 200 g Reis\n- 2 Paprika\n\n## Zubereitung\n\n1. Kochen.`
          : 'ok';
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ choices: [{ message: { content } }] }));
      });
    }).listen(0, '127.0.0.1', () => resolve(server));
  });
});
test.afterAll(() => fakeAI?.close());

test('generate a recipe: wishes, roll again, change wishes, accept', async ({ page }) => {
  await signIn(page, { sub: 'gina', name: 'Gina', admin: true });
  const space = unique('Generiert');
  await createSpace(page, space);
  await useAsCookbook(page, space);
  await page.goto('/admin/ai');
  await page.getByLabel('KI eingeschaltet').check();
  await page.getByLabel('Basis-URL').fill(`http://127.0.0.1:${fakeAI.address().port}/v1`);
  await page.getByLabel('Modell').fill('test-model');
  await page.getByRole('button', { name: 'Speichern und Verbindung testen' }).click();
  await expect(page.getByRole('status')).toHaveText('Verbindung steht: Die KI hat geantwortet.');

  try {
    await page.goto('/recipes');
    await page.getByRole('link', { name: 'Rezept generieren' }).click();
    await page.getByLabel('Zutaten', { exact: true }).fill('Reis, Paprika');
    await page.getByLabel('Richtung').selectOption({ label: 'Gutbürgerlich' });
    await page.getByRole('button', { name: 'Rezept generieren' }).click();
    const first = await page.getByLabel('Titel').inputValue();
    expect(first).toMatch(/^Pfanne Nummer \d+$/);

    await page.getByRole('button', { name: 'Neu würfeln' }).click();
    await expect(page.getByLabel('Titel')).not.toHaveValue(first);

    await page.getByRole('link', { name: 'Wünsche ändern' }).click();
    await expect(page.getByLabel('Zutaten', { exact: true })).toHaveValue('Reis, Paprika');
    await expect(page.getByLabel('Richtung')).toHaveValue('homestyle');
    await page.getByRole('button', { name: 'Rezept generieren' }).click();
    const title = unique(await page.getByLabel('Titel').inputValue());
    await page.getByLabel('Titel').fill(title);
    await page.getByRole('button', { name: 'Übernehmen' }).click();
    await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}\/edit$/);

    await page.goto('/recipes');
    await page.locator('.tag-filter').getByRole('link', { name: 'ki-generiert' }).click();
    await expect(page.locator('.recipe-list').getByRole('link', { name: title })).toBeVisible();
  } finally {
    await page.goto('/admin/ai');
    await page.getByLabel('KI eingeschaltet').uncheck();
    await page.getByRole('button', { name: 'Speichern', exact: true }).click();
  }
});
