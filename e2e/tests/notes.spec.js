// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace } from './helpers.js';

test('notes: write one, find it in the list, delete it', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  const text = unique('Zählerstand Keller');
  await page.goto('/notes/new');
  await page.getByLabel('Notiz', { exact: true }).fill(`${text}\n4711 kWh`);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Gespeichert');
  await expect(page.getByLabel('Notiz', { exact: true })).toHaveValue('');

  await page.getByRole('link', { name: 'Alle Notizen' }).click();
  await page.getByRole('link', { name: text }).click();
  await expect(page.getByRole('heading', { level: 1, name: text })).toBeVisible();
  await expect(page.locator('.prose')).toContainText('4711 kWh');

  await page.getByRole('link', { name: 'Notiz löschen …' }).click();
  await page.getByRole('button', { name: `„${text}“ löschen` }).click();
  await expect(page).toHaveURL(/\/notes$/);
  await expect(page.getByRole('link', { name: text })).toHaveCount(0);
});

test('notes: an empty note is refused with a reason', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  await page.goto('/notes/new');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Schreib etwas oder hänge ein Foto an.');
});

// A tiny valid PNG (1×1 pixel).
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64',
);

test('notes: edit a note and add a photo', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  const text = unique('Heizung ablesen');
  await page.goto('/notes/new');
  await page.getByLabel('Notiz', { exact: true }).fill(text);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Gespeichert');

  await page.goto('/notes');
  await page.getByRole('link', { name: text }).click();
  await page.getByRole('link', { name: 'Bearbeiten' }).click();
  await page.locator('#file').setInputFiles({ name: 'Zähler.png', mimeType: 'image/png', buffer: PNG });
  await expect(page.getByRole('textbox', { name: 'Quelltext' })).toHaveValue(/!\[Zähler\]\(attachments\/zaehler\.png\)/);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/notes\/[a-z2-7]{12}$/);
  await expect(page.locator('.prose img')).toHaveAttribute('src', /\/notes\/[a-z2-7]{12}\/attachments\/zaehler\.png$/);
});

test('notes: turn a note into a wiki page', async ({ page }) => {
  await signIn(page, { sub: 'nils', name: 'Nils', admin: true });
  const space = unique('Notizziel');
  await createSpace(page, space);
  const text = unique('Rasen düngen');
  await page.goto('/notes/new');
  await page.getByLabel('Notiz', { exact: true }).fill(`${text}\nIm April und im September.`);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Gespeichert');

  await page.goto('/notes');
  await page.getByRole('link', { name: text }).click();
  await page.getByRole('link', { name: 'Als Wiki-Seite anlegen' }).click();
  await expect(page.getByLabel('Titel der Seite')).toHaveValue(text);
  await page.getByLabel('Wo').selectOption({ label: space });
  await page.getByRole('button', { name: 'Als Wiki-Seite anlegen' }).click();

  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);
  await expect(page.getByRole('heading', { level: 1, name: text })).toBeVisible();
  await expect(page.locator('.prose')).toContainText('Im April und im September.');
  await page.goto('/notes');
  await expect(page.getByRole('link', { name: text })).toHaveCount(0);
});

test('notes: capture with a photo from the library', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  const text = unique('Typenschild');
  await page.goto('/notes/new');
  await page.getByLabel('Notiz', { exact: true }).fill(text);
  await page.locator('#pick').setInputFiles({ name: 'schild.png', mimeType: 'image/png', buffer: PNG });
  await expect(page.locator('#previews img')).toHaveCount(1);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Gespeichert');
  await expect(page.locator('#previews img')).toHaveCount(0);

  await page.goto('/notes');
  await page.getByRole('link', { name: text }).click();
  await expect(page.locator('.prose img')).toHaveAttribute('src', /attachments\/schild\.png$/);
});

test('notes: the actions of a note share one height', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  const text = unique('Gleich hoch');
  await page.goto('/notes/new');
  await page.getByLabel('Notiz', { exact: true }).fill(text);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Gespeichert');
  await page.goto('/notes');
  await page.getByRole('link', { name: text }).click();
  const heights = await page.locator('.actions .btn').evaluateAll((els) => els.map((e) => e.getBoundingClientRect().height));
  expect(heights).toHaveLength(3);
  expect(new Set(heights).size).toBe(1);
});
