// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

test('sign in shows the start page in German', async ({ page }) => {
  await signIn(page);
  await expect(page.getByRole('heading', { level: 1, name: 'Wiki-Bereiche' })).toBeVisible();
  await expect(page.locator('html')).toHaveAttribute('lang', 'de');
});

test('language can be switched to English', async ({ page }) => {
  await signIn(page, { sub: 'ben', name: 'Ben' });
  await page.getByText('Ben', { exact: true }).click();
  await page.getByLabel('Sprache').selectOption('en');
  await page.getByRole('button', { name: 'Übernehmen' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Wiki spaces' })).toBeVisible();
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

test('navigation starts folded on phones and open on wide screens', async ({ page, isMobile }) => {
  await signIn(page, { sub: 'fritz', name: 'Fritz' });
  const nav = page.locator('.sidenav > details');
  if (isMobile) {
    await expect(nav).not.toHaveAttribute('open', '');
    await page.locator('.sidenav summary').click();
    await expect(nav).toHaveAttribute('open', '');
  } else {
    await expect(nav).toHaveAttribute('open', '');
  }
});

test('links styled as buttons are not underlined', async ({ page }) => {
  await signIn(page, { sub: 'gerd', name: 'Gerd', admin: true });
  const button = page.getByRole('link', { name: 'Bereich anlegen' });
  await expect(button).toHaveCSS('text-decoration-line', 'none');
});

test('dev login switches between made-up people', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('button', { name: 'Ben', exact: true }).click();
  const menu = page.locator('.usermenu summary');
  await expect(menu).toHaveText('Ben');
  await menu.click();
  await page.getByRole('link', { name: 'Person wechseln' }).click();
  await page.getByRole('button', { name: 'Oma Gerda', exact: true }).click();
  await expect(menu).toHaveText('Oma Gerda');
  await expect(page.getByRole('heading', { level: 1, name: 'Wiki-Bereiche' })).toBeVisible();
});

test('a short page keeps the closed menu compact on phones', async ({ page, isMobile }) => {
  test.skip(!isMobile, 'the menu sits above the content only on phones');
  await signIn(page, { sub: 'kurt', name: 'Kurt' });
  await page.goto('/');
  const nav = await page.locator('.sidenav').boundingBox();
  expect(nav.height).toBeLessThan(120);
});

const PIXEL = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64',
);

test('the save button stays visible with a photo attached', async ({ page, isMobile }) => {
  await signIn(page, { sub: 'kurt', name: 'Kurt' });
  await page.goto('/notes/new');
  await page.locator('#pick').setInputFiles({ name: 'a.png', mimeType: 'image/png', buffer: PIXEL });
  await expect(page.locator('#previews img')).toHaveCount(1);
  const save = await page.getByRole('button', { name: 'Speichern' }).boundingBox();
  const rail = await page.locator('.rail').boundingBox();
  const bottom = isMobile ? rail.y : page.viewportSize().height; // phones: the bar sits at the bottom
  expect(save.y + save.height).toBeLessThanOrEqual(bottom);
});
