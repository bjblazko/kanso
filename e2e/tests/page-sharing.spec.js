// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, createPage } from './helpers.js';

test('a page shared on its own: the other person finds it and nothing else of the space', async ({ page }) => {
  const christa = unique('Christa');
  await signIn(page, { sub: 'christa', name: christa }); // known to Kanso from now on
  await page.context().clearCookies();
  await signIn(page, { sub: 'timo', name: 'Timo', admin: true });

  const space = unique('Timos Bereich');
  await createSpace(page, space);
  await createPage(page, 'Wunschliste');
  await createPage(page, 'Bücher', { child: true });
  await page.getByRole('link', { name: space }).first().click();
  await createPage(page, 'Geheim');

  await page.getByRole('link', { name: space }).first().click();
  await page.locator('main').getByRole('link', { name: 'Wunschliste' }).click();
  await page.locator('.audience').getByRole('link', { name: 'Freigabe ändern' }).click();
  await expect(page.getByText('Gilt auch für 1 Unterseite.')).toBeVisible();
  await page.getByLabel('Wer').selectOption({ label: christa });
  await page.getByLabel('Rolle').selectOption({ label: 'Lesen' });
  await page.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect(page.locator('.members')).toContainText(christa);

  await page.context().clearCookies();
  await signIn(page, { sub: 'christa', name: christa });
  await page.goto('/');
  const main = page.locator('main');
  await expect(main.getByRole('heading', { name: 'Für dich freigegeben' })).toBeVisible();
  await main.getByRole('link', { name: 'Wunschliste' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Wunschliste' })).toBeVisible();
  await expect(main.getByRole('link', { name: 'Bücher' })).toBeVisible();
  await expect(page.getByText('Geheim')).toHaveCount(0);
  await expect(page.getByRole('link', { name: space })).toHaveCount(0);
});

test('a page moves to another space, after saying who will see it then', async ({ page }) => {
  await signIn(page, { sub: 'mo', name: 'Mo', admin: true });
  const from = unique('Haus alt');
  const to = unique('Keller neu');
  await createSpace(page, to);
  await createSpace(page, from);
  await page.getByRole('link', { name: 'Einstellungen' }).click(); // someone reads "from", nobody "to"
  await page.getByLabel('Wer').selectOption({ label: 'Mo' });
  await page.getByRole('button', { name: 'Hinzufügen' }).click();
  await page.getByRole('link', { name: 'Zum Bereich' }).click();
  await createPage(page, 'Heizung');
  await createPage(page, 'Wartung', { child: true });
  await page.getByRole('link', { name: 'Heizung' }).first().click();

  await page.locator('details.more summary').click();
  await page.getByLabel('Verschieben nach').selectOption({ label: to });
  await page.getByRole('button', { name: 'Seite verschieben' }).click();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText(`„Heizung“ nach „${to}“ verschieben?`);
  await page.getByRole('button', { name: 'Verschieben', exact: true }).click();

  await expect(page.locator('.breadcrumb')).toContainText(to);
  await expect(page.locator('main').getByRole('link', { name: 'Wartung' })).toBeVisible();
});
