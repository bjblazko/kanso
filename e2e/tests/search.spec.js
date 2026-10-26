// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, createPage } from './helpers.js';

test('search finds pages by title and text, with umlauts either way', async ({ page }) => {
  const word = `Zählerstand${test.info().project.name.replace(/[^a-z]/g, '')}`;
  await signIn(page, { sub: 'suse', name: 'Suse', admin: true });
  await createSpace(page, unique('Haushalt'));
  await createPage(page, 'Strom');
  await page.getByRole('link', { name: 'Bearbeiten' }).click();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill(`Den ${word} jeden Monat notieren.`);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);

  const box = page.getByRole('searchbox', { name: 'Suchen' });
  await box.fill(word.replace('ä', 'ae').slice(0, -2)); // umlaut spelled out, last word as prefix
  await box.press('Enter');
  await expect(page.getByRole('heading', { level: 1, name: 'Suche' })).toBeVisible();
  await expect(page.locator('.results').getByRole('link', { name: 'Strom' })).toBeVisible();
  await expect(page.locator('.snippet')).toContainText(word);

  await box.fill('gibtesnichtxyz');
  await box.press('Enter');
  await expect(page.getByText('Nichts gefunden für „gibtesnichtxyz“.')).toBeVisible();
});
