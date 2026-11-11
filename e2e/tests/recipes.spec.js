// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace } from './helpers.js';

test('recipe: create, find by tag, cook with more servings, tick off', async ({ page }) => {
  await signIn(page, { sub: 'rita', name: 'Rita', admin: true });
  const space = unique('Rezepte');
  await createSpace(page, space);

  await page.goto('/recipes');
  await page.getByRole('link', { name: 'Neues Rezept' }).click();
  const title = unique('Kartoffelcurry');
  await page.getByLabel('Titel').fill(title);
  await page.getByLabel('Wiki-Bereich').selectOption({ label: space });
  await page.getByRole('button', { name: 'Rezept anlegen' }).click();

  await expect(page.getByLabel('Ist ein Rezept')).toBeChecked();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill(
    'Mild.\n\n## Zutaten\n\n- 800 g Kartoffeln\n- 2 Zwiebeln\n\n## Zubereitung\n\n1. Schälen.\n2. Kochen.\n');
  await page.getByLabel('Portionen').fill('4');
  await page.getByLabel('Tags').fill('Deftig');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);

  await page.goto('/recipes');
  await page.locator('.tag-filter').getByRole('link', { name: 'deftig' }).click();
  await expect(page).toHaveURL(/tag=deftig/);
  await page.locator('.recipe-list').getByRole('link', { name: title }).click();

  await page.getByRole('link', { name: 'Kochen' }).click();
  await page.getByRole('link', { name: 'Eine Portion mehr' }).click();
  await page.getByRole('link', { name: 'Eine Portion mehr' }).click();
  await expect(page.locator('.servings-n')).toHaveText('6');
  await expect(page.locator('.cook-list')).toContainText('1200 g Kartoffeln');

  const first = page.locator('.cook-list input[type="checkbox"]').first();
  await first.check();
  await page.reload();
  await expect(page.locator('.cook-list input[type="checkbox"]').first()).toBeChecked();

  await page.getByRole('button', { name: 'Häkchen entfernen' }).click();
  await page.reload();
  await expect(page.locator('.cook-list input[type="checkbox"]').first()).not.toBeChecked();
});

test('recipe: give stars, filter the kitchen by them, clear them', async ({ page }) => {
  await signIn(page, { sub: 'rita', name: 'Rita', admin: true });
  const space = unique('Sterne');
  await createSpace(page, space);
  await page.goto('/recipes/new');
  const title = unique('Linsensuppe');
  await page.getByLabel('Titel').fill(title);
  await page.getByLabel('Wiki-Bereich').selectOption({ label: space });
  await page.getByRole('button', { name: 'Rezept anlegen' }).click();
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);

  const stars = page.getByRole('group', { name: 'Noch nicht bewertet' });
  await stars.getByRole('button', { name: 'Mit 5 bewerten' }).click();
  await expect(page.getByRole('group', { name: '5 von 5 Sternen' })).toBeVisible();

  await page.goto('/recipes');
  await page.locator('.tag-filter').getByRole('link', { name: '5 Sterne' }).click();
  await expect(page).toHaveURL(/stars=5/);
  const row = page.locator('.recipe-list li', { hasText: title });
  await expect(row.getByRole('img', { name: '5 von 5 Sternen' })).toBeVisible();

  await row.getByRole('link', { name: title }).click();
  await page.getByRole('button', { name: 'Mit 5 bewerten' }).click(); // the same stars again clear them
  await expect(page.getByRole('group', { name: 'Noch nicht bewertet' })).toBeVisible();
});
