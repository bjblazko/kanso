// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, createSpace, createPage, unique } from './helpers.js';

test('Berichte: the overview counts, draws and switches scope', async ({ page }) => {
  await signIn(page, { sub: 'rolf', name: 'Rolf', admin: true });
  await createSpace(page, unique('Berichtsraum'));
  await createPage(page, unique('Zählseite'));

  await page.getByLabel('Menü von Rolf').click();
  await page.getByRole('link', { name: 'Berichte' }).click();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Berichte');
  await expect(page.getByRole('link', { name: 'Alle', exact: true })).toHaveAttribute('aria-current', 'page');

  const card = page.locator('section', { has: page.getByRole('heading', { name: 'Bestand nach Typ' }) });
  await expect(card.locator('svg')).toBeVisible();

  await page.getByRole('link', { name: 'Eigene', exact: true }).click();
  await expect(page.getByRole('link', { name: 'Eigene', exact: true })).toHaveAttribute('aria-current', 'page');
  const own = page.locator('section', { has: page.getByRole('heading', { name: 'Bestand nach Typ' }) });
  await own.getByText('Als Tabelle').click();
  await expect(own.getByRole('row', { name: /Seiten/ })).toBeVisible();

  await page.getByRole('link', { name: 'Speicher', exact: true }).click();
  const areas = page.locator('section', { has: page.getByRole('heading', { name: 'Wo der Platz steckt' }) });
  await expect(areas.locator('svg')).toBeVisible();
  await expect(page.getByRole('link', { name: 'Eigene', exact: true })).toHaveAttribute('aria-current', 'page');

  await page.getByRole('link', { name: 'Aktivität', exact: true }).click();
  const cal = page.locator('section', { has: page.getByRole('heading', { name: 'Aktivitätskalender' }) });
  await expect(cal.locator('svg')).toBeVisible();

  await page.locator('main').getByRole('link', { name: 'Wiki', exact: true }).click();
  const age = page.locator('section', { has: page.getByRole('heading', { name: 'Wie alt ist das Wiki?' }) });
  await expect(age.locator('svg')).toBeVisible();

  await page.locator('main').getByRole('link', { name: 'Aufgaben', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Eisenhower-Verteilung' })).toBeVisible();

  await page.locator('main').getByRole('link', { name: 'Küche', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Meistgekocht' })).toBeVisible();
});
