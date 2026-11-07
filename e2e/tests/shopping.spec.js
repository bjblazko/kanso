// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace } from './helpers.js';

const item = (page, text) => page.locator('.todo-item', { has: page.getByRole('checkbox', { name: text }) });

test('shopping list: amounts with − and + in sensible steps, a note, filled from a recipe', async ({ page }) => {
  await signIn(page, { sub: 'sina', name: 'Sina', admin: true });
  const name = unique('Wocheneinkauf');
  await page.goto('/todos');
  await page.getByLabel('Neue Liste').fill(name);
  await page.getByRole('button', { name: 'Einkaufsliste anlegen' }).click();
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();

  const add = async (text) => {
    await page.getByLabel('Was fehlt?').fill(text);
    await page.getByLabel('Was fehlt?').press('Enter');
    await expect(page.getByRole('checkbox', { name: text })).toBeVisible();
  };
  const step = async (text, which) => item(page, text).getByRole('button', { name: which }).click();
  const shows = (text) => expect(page.getByRole('checkbox', { name: text })).toBeVisible();

  await add('2 Zitronen');
  await step('2 Zitronen', 'Eins mehr');
  await shows('3 Zitronen');
  await step('3 Zitronen', 'Eins weniger');
  await step('2 Zitronen', 'Eins weniger');
  await expect(item(page, '1 Zitronen').getByRole('button', { name: 'Eins weniger' })).toBeDisabled();

  await add('250 g Butter');
  await step('250 g Butter', 'Eins mehr');
  await shows('300 g Butter');
  await add('87 g Parmesan');
  await step('87 g Parmesan', 'Eins mehr');
  await shows('90 g Parmesan');
  await step('90 g Parmesan', 'Eins weniger');
  await shows('80 g Parmesan');
  await step('300 g Butter', 'Menge ändern');
  await page.getByLabel('Menge', { exact: true }).fill('275');
  await page.getByLabel('Menge', { exact: true }).press('Enter');
  await shows('275 g Butter');
  await add('1,5 kg Mehl');
  await step('1,5 kg Mehl', 'Eins mehr');
  await shows('2 kg Mehl');

  await add('1 l Milch');
  await page.getByRole('button', { name: '1 l Milch', exact: true }).click();
  await page.getByLabel('Notiz').fill('laktosefrei, sonst Hafermilch');
  await page.getByLabel('Notiz').press('Enter');
  await expect(item(page, '1 l Milch')).toContainText('laktosefrei, sonst Hafermilch');
  await expect(page.locator('#todos-status')).toHaveText(''); // synced, nothing waiting
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
