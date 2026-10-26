// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

test('sign in shows the start page in German', async ({ page }) => {
  await signIn(page);
  await expect(page.getByRole('heading', { level: 1, name: 'Bereiche' })).toBeVisible();
  await expect(page.locator('html')).toHaveAttribute('lang', 'de');
});

test('language can be switched to English', async ({ page }) => {
  await signIn(page, { sub: 'ben', name: 'Ben' });
  await page.getByText('Ben', { exact: true }).click();
  await page.getByLabel('Sprache').selectOption('en');
  await page.getByRole('button', { name: 'Übernehmen' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Spaces' })).toBeVisible();
});

test('licenses page lists shipped third-party work', async ({ page }) => {
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.getByRole('link', { name: 'Lizenzen und Dank' }).click();
  await expect(page.getByRole('link', { name: 'IBM Plex Sans and IBM Plex Mono' })).toBeVisible();
  await expect(page.getByText('go.yaml.in/yaml/v3')).toBeVisible();
});

test('unknown pages say so', async ({ page }) => {
  await signIn(page, { sub: 'dora', name: 'Dora' });
  const response = await page.goto('/gibt/es/nicht');
  expect(response.status()).toBe(404);
  await expect(page.getByRole('heading', { name: 'Diese Seite gibt es nicht.' })).toBeVisible();
});

test('layout has no horizontal scroll', async ({ page }) => {
  await signIn(page, { sub: 'erik', name: 'Erik' });
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  expect(overflow).toBeLessThanOrEqual(0);
});
