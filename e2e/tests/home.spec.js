// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, createPage } from './helpers.js';

test('start page: what happened, with charts, window and own changes', async ({ page }) => {
  await signIn(page, { sub: 'hanna', name: 'Hanna', admin: true });
  const space = unique('Neuigkeiten');
  await createSpace(page, space);
  const title = unique('Kellerplan');
  await createPage(page, title);

  await page.goto('/');
  const summary = page.locator('.home-summary');
  await expect(summary.getByRole('heading', { name: 'Was passiert ist' })).toBeVisible();
  await expect(summary.getByRole('link', { name: title })).toHaveCount(0); // own changes hidden at first

  await summary.getByRole('link', { name: 'Eigene einblenden' }).click();
  await expect(page).toHaveURL(/own=1/);
  await expect(summary.locator('.home-sentence')).toContainText('neue Seite');
  await expect(summary.getByRole('link', { name: title })).toBeVisible();
  await expect(summary.locator('.report-chart[data-chart="spark"] svg').first()).toBeVisible();
  await expect(summary.locator('[data-id="band"] svg')).toBeVisible();
  await expect(summary.locator('[data-id="people"] svg')).toBeVisible();

  await summary.getByLabel('Zeitraum').selectOption({ label: '7 Tage' });
  await summary.getByRole('button', { name: 'Anzeigen' }).click();
  await expect(page).toHaveURL(/range=7d/);
  await expect(page).toHaveURL(/own=1/);
  await expect(summary.getByRole('link', { name: 'Eigene ausblenden' })).toBeVisible();

  await page.setViewportSize({ width: 390, height: 800 });
  await page.reload();
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
});
