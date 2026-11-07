// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace } from './helpers.js';

test('meal plan: plan two meals, shop for both days, take one off again', async ({ page }) => {
  const me = unique('Mia');
  await signIn(page, { sub: 'mia', name: me, admin: true });
  const group = unique('Küchengruppe');
  await page.goto('/admin/groups');
  await page.getByLabel('Name der Gruppe').fill(group);
  await page.getByRole('button', { name: 'Gruppe anlegen' }).click();
  const section = page.locator('section.group', { has: page.getByRole('heading', { name: group }) });
  await section.getByLabel('Person').selectOption({ label: me });
  await section.getByRole('button', { name: 'Hinzufügen' }).click();

  const planName = unique('Familienküche');
  await page.goto('/admin/mealplans');
  await page.getByLabel('Name', { exact: true }).fill(planName);
  await page.getByLabel('Gruppe').selectOption(group);
  await page.getByRole('button', { name: 'Kochplan anlegen' }).click();

  await createSpace(page, unique('Rezepte'));
  await page.goto('/recipes/new');
  const recipe = unique('Linsensuppe');
  await page.getByLabel('Titel').fill(recipe);
  await page.getByRole('button', { name: 'Rezept anlegen' }).click();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill('## Zutaten\n\n- 250 g Linsen\n- 1 Zwiebel\n');
  await page.getByLabel('Portionen').fill('2');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);
  const recipeURL = page.url();

  for (const day of ['2026-11-16', '2026-11-18']) {
    await page.goto(recipeURL);
    await page.getByRole('link', { name: 'Einplanen' }).click();
    await page.getByLabel('In welchen Kochplan?').selectOption({ label: planName });
    await page.getByLabel('Tag').fill(day);
    await page.getByRole('button', { name: 'Einplanen' }).click();
    await expect(page.getByRole('heading', { level: 1, name: planName })).toBeVisible();
  }
  await expect(page.locator('.week-meals a', { hasText: recipe })).toHaveCount(2);
  const weekURL = page.url();

  await page.getByRole('checkbox', { name: /Mo 16\.11\./ }).check();
  await page.getByRole('checkbox', { name: /Mi 18\.11\./ }).check();
  await page.getByRole('button', { name: 'Einkaufen' }).click();
  await expect(page.locator('.shop-meal')).toHaveCount(2);
  await expect(page.getByRole('button', { name: 'Vorschau aktualisieren' })).toBeHidden();
  await expect(page.locator('.shop-meal').nth(1)).toContainText('250 g Linsen → 500 g Linsen');
  await page.getByRole('button', { name: 'Auf die Liste' }).click();
  await expect(page.getByRole('checkbox', { name: '500 g Linsen' })).toBeVisible();

  await page.goto(weekURL);
  await expect(page.locator('.week-meals')).toHaveCount(2);
  await page.getByRole('link', { name: `Aus dem Plan nehmen: ${recipe}` }).last().click();
  await page.getByRole('button', { name: 'Entfernen und Zutaten abziehen' }).click();
  await expect(page.locator('.week-meals a', { hasText: recipe })).toHaveCount(1);
  await page.goto('/todos');
  await page.getByRole('link', { name: 'Einkauf' }).last().click();
  await expect(page.getByRole('checkbox', { name: '250 g Linsen' })).toBeVisible();
});
