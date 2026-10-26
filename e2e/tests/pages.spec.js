// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, createPage } from './helpers.js';

test('pages: create, nest, move and delete', async ({ page }) => {
  await signIn(page, { sub: 'pia', name: 'Pia', admin: true });
  await createSpace(page, unique('Wiki'));

  await createPage(page, 'Über uns');
  await expect(page.getByRole('heading', { level: 1, name: 'Über uns' })).toBeVisible();
  await createPage(page, 'Größe & Gewicht', { child: true });
  await expect(page.getByRole('navigation', { name: 'Pfad' }).getByRole('link', { name: 'Über uns' })).toBeVisible();

  // The navigation shows the tree with the current page marked.
  const nav = page.getByRole('navigation', { name: 'Bereiche' });
  await expect(nav.locator('a[aria-current="page"]')).toHaveText('Größe & Gewicht');

  // Move the sub-page to the top level.
  await page.getByText('Verschieben oder löschen').click();
  await page.getByLabel('Verschieben nach').selectOption({ label: unique('Wiki') });
  await page.getByRole('button', { name: 'Seite verschieben' }).click();
  await expect(page.getByRole('navigation', { name: 'Pfad' }).getByRole('link', { name: 'Über uns' })).toHaveCount(0);

  // Delete needs a second, explicit step that names the page.
  await page.getByText('Verschieben oder löschen').click();
  await page.getByRole('link', { name: 'Seite löschen …' }).click();
  await page.getByRole('button', { name: '„Größe & Gewicht“ löschen' }).click();
  await expect(page.getByRole('heading', { level: 1 })).toContainText(unique('Wiki'));
  await expect(page.getByRole('main').getByRole('link', { name: 'Größe & Gewicht' })).toHaveCount(0);
});

test('pages: a title is required', async ({ page }) => {
  await signIn(page, { sub: 'quinn', name: 'Quinn', admin: true });
  await createSpace(page, unique('Leer'));
  await page.getByRole('link', { name: 'Seite anlegen' }).click();
  await page.getByLabel('Titel').fill('   ');
  await page.getByRole('button', { name: 'Seite anlegen' }).click();
  await expect(page.getByText('Gib der Seite einen Titel.')).toBeVisible();
});
