// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, createPage } from './helpers.js';

async function edit(page, text) {
  await page.getByRole('link', { name: 'Bearbeiten' }).click();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill(text);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);
}

test('history: compare an older version and restore it', async ({ page }) => {
  await signIn(page, { sub: 'hugo', name: 'Hugo', admin: true });
  await createSpace(page, unique('Verlauf'));
  await createPage(page, 'Rezept');
  await edit(page, 'Mehl 500 g\n');
  await edit(page, 'Mehl 750 g\n');

  await page.getByRole('link', { name: 'Verlauf', exact: true }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Verlauf' })).toBeVisible();
  await expect(page.locator('.history li')).toHaveCount(3);
  await page.locator('.history li').nth(1).getByRole('link').click();

  await expect(page.locator('.diff-del')).toContainText('Mehl 500 g');
  await expect(page.locator('.diff-add')).toContainText('Mehl 750 g');
  await page.getByRole('button', { name: 'Diese Fassung wiederherstellen' }).click();
  await expect(page.locator('.prose')).toContainText('Mehl 500 g');
});
