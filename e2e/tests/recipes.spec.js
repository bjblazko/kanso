// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, useAsCookbook } from './helpers.js';

test('recipe: create, find by tag, cook with more servings, tick off', async ({ page }) => {
  await signIn(page, { sub: 'rita', name: 'Rita', admin: true });
  const space = unique('Rezepte');
  await createSpace(page, space);
  await useAsCookbook(page, space);

  await page.goto('/recipes');
  await page.getByRole('link', { name: 'Neues Rezept' }).click();
  const title = unique('Kartoffelcurry');
  await page.getByLabel('Titel').fill(title);
  await page.getByRole('button', { name: 'Rezept anlegen' }).click();

  await expect(page.getByLabel('Ist ein Rezept')).toBeChecked();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill(
    'Mild.\n\n## Zutaten\n\n- 800 g Kartoffeln\n- 2 Zwiebeln\n\n## Zubereitung\n\n1. Schälen.\n2. Kochen.\n');
  await page.getByLabel('Portionen').fill('4');
  await page.getByLabel('Tags').fill('Deftig');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);

  // A recipe lives in the kitchen: no spaces bar, back to the recipes, not in the space's tree.
  await expect(page.locator('nav.rail').getByRole('link', { name: 'Küche' })).toHaveAttribute('aria-current', 'true');
  await expect(page.locator('.breadcrumb').getByRole('link', { name: 'Rezepte' })).toBeVisible();
  await page.goto('/wiki');
  await expect(page.locator('main').getByRole('link', { name: space, exact: true })).toHaveCount(0); // the cookbook is not in the wiki

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

  const bar = page.getByRole('progressbar', { name: 'Fortschritt' });
  const percent = page.locator('.cook-body .todo-percent');
  await expect(bar).toHaveAttribute('aria-valuenow', '0');
  await page.getByRole('checkbox', { name: 'Schälen.' }).check();
  await expect(bar).toHaveAttribute('aria-valuenow', '50');
  await expect(percent).toHaveText('50 %');
  await page.reload();
  await expect(percent).toHaveText('50 %');
  await page.getByRole('checkbox', { name: 'Kochen.' }).check();
  await expect(bar).toHaveAttribute('aria-valuenow', '100');
  await expect(bar).toHaveAttribute('data-state', 'full');
  await page.getByRole('button', { name: 'Häkchen entfernen' }).click();
  await expect(percent).toHaveText('0 %');
});

test('recipe: give stars, filter the kitchen by them, clear them', async ({ page }) => {
  await signIn(page, { sub: 'rita', name: 'Rita', admin: true });
  const space = unique('Sterne');
  await createSpace(page, space);
  await useAsCookbook(page, space);
  await page.goto('/recipes/new');
  const title = unique('Linsensuppe');
  await page.getByLabel('Titel').fill(title);
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
