// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, createPage } from './helpers.js';

// 1×1 transparent PNG
const PNG = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=', 'base64');

async function showPreview(page) {
  const tab = page.getByRole('tab', { name: 'Vorschau' });
  if (await tab.isVisible()) await tab.click();
}

async function showSource(page) {
  const tab = page.getByRole('tab', { name: 'Quelltext' });
  if (await tab.isVisible()) await tab.click();
}

test('editor: write Markdown, see preview, attach an image, save', async ({ page }) => {
  await signIn(page, { sub: 'ella', name: 'Ella', admin: true });
  await createSpace(page, unique('Editor'));
  await createPage(page, 'Heizung');
  await page.getByRole('link', { name: 'Bearbeiten' }).click();

  const source = page.getByRole('textbox', { name: 'Quelltext' });
  await source.fill('# Wartung\n\n**Jährlich** den Filter prüfen.\n');
  await showPreview(page);
  await expect(page.locator('#preview strong')).toHaveText('Jährlich');

  await showSource(page);
  await page.locator('#file').setInputFiles({ name: 'Filter Foto.png', mimeType: 'image/png', buffer: PNG });
  await expect(source).toHaveValue(/!\[Filter Foto\]\(attachments\/filter-foto\.png\)/);

  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);
  await expect(page.locator('.prose strong')).toHaveText('Jährlich');
  await expect(page.locator('.prose img')).toHaveAttribute('src', /\/attachments\/filter-foto\.png$/);
  await expect(page.getByRole('link', { name: 'filter-foto.png' })).toBeVisible();
});

test('editor: second person sees the lock; after a break the text survives', async ({ browser }) => {
  const project = test.info().project.name;
  const ownerCtx = await browser.newContext({ locale: 'de-DE' });
  const editorCtx = await browser.newContext({ locale: 'de-DE' });
  const owner = await ownerCtx.newPage();
  const editor = await editorCtx.newPage();

  await signIn(editor, { sub: 'emil', name: 'Emil' });
  await signIn(owner, { sub: 'olga', name: 'Olga', admin: true });
  await createSpace(owner, unique('Sperren'));
  await owner.getByRole('link', { name: 'Einstellungen' }).click();
  await owner.getByLabel('Wer', { exact: true }).selectOption(`emil-${project}`);
  await owner.getByLabel('Rolle', { exact: true }).selectOption('editor');
  await owner.getByRole('button', { name: 'Hinzufügen' }).click();
  await owner.getByRole('link', { name: 'Zum Bereich' }).click();
  await createPage(owner, 'Gemeinsam');
  const pageURL = owner.url();

  await editor.goto(pageURL);
  await editor.getByRole('link', { name: 'Bearbeiten' }).click();
  await editor.getByRole('textbox', { name: 'Quelltext' }).fill('Mein wichtiger Text');

  await owner.goto(pageURL + '/edit');
  await expect(owner.getByText('Emil bearbeitet diese Seite gerade.')).toBeVisible();
  await owner.getByRole('button', { name: 'Sperre aufheben' }).click();
  await expect(owner.getByRole('textbox', { name: 'Quelltext' })).toBeVisible();

  await editor.getByRole('button', { name: 'Speichern' }).click();
  await expect(editor.getByRole('alert')).toContainText('Die Sperre ist abgelaufen oder wurde aufgehoben.');
  await expect(editor.getByRole('textbox', { name: 'Quelltext' })).toHaveValue('Mein wichtiger Text');

  await ownerCtx.close();
  await editorCtx.close();
});
