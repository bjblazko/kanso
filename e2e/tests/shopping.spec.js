// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace } from './helpers.js';

const item = (page, text) => page.locator('.todo-item', { has: page.getByRole('checkbox', { name: text }) });

test('shopping list: amounts with − and +, a note, filled from a recipe', async ({ page }) => {
  await signIn(page, { sub: 'sina', name: 'Sina', admin: true });
  const name = unique('Wocheneinkauf');
  await page.goto('/todos');
  await page.getByLabel('Neue Liste').fill(name);
  await page.getByRole('button', { name: 'Einkaufsliste anlegen' }).click();
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();

  await page.getByLabel('Was fehlt?').fill('2 l Milch');
  await page.getByLabel('Was fehlt?').press('Enter');
  await expect(page.getByRole('checkbox', { name: '2 l Milch' })).toBeVisible();
  await item(page, '2 l Milch').getByRole('button', { name: 'Eins mehr' }).click();
  await expect(page.getByRole('checkbox', { name: '3 l Milch' })).toBeVisible();
  await item(page, '3 l Milch').getByRole('button', { name: 'Eins weniger' }).click();
  await item(page, '2 l Milch').getByRole('button', { name: 'Eins weniger' }).click();
  await expect(item(page, '1 l Milch').getByRole('button', { name: 'Eins weniger' })).toBeDisabled();

  await page.getByRole('button', { name: '1 l Milch' }).click();
  await page.getByLabel('Notiz').fill('laktosefrei, sonst Hafermilch');
  await page.getByLabel('Notiz').press('Enter');
  await page.reload();
  await expect(item(page, '1 l Milch')).toContainText('laktosefrei, sonst Hafermilch');

  await createSpace(page, unique('Rezepte'));
  await page.goto('/recipes/new');
  const recipe = unique('Milchreis');
  await page.getByLabel('Titel').fill(recipe);
  await page.getByRole('button', { name: 'Rezept anlegen' }).click();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill('## Zutaten\n\n- 1 l Milch\n- 250 g Milchreis\n');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.getByRole('link', { name: 'Einkaufen' }).click();
  await page.getByLabel('Auf welche Einkaufsliste?').selectOption({ label: name });
  await expect(page.locator('.shop-lines')).toContainText('1 l Milch → 2 l Milch');
  await page.getByRole('button', { name: 'Auf die Liste' }).click();

  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();
  await expect(item(page, '2 l Milch')).toContainText(`für ${recipe}`);
  await expect(item(page, '250 g Milchreis')).toContainText(`für ${recipe}`);
});
