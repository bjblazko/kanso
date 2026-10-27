// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique } from './helpers.js';

// Playwright's WebKit does not run service workers reliably; the offline
// behaviour is checked in Chromium (and by hand on an iPhone).
test.skip(({ browserName }) => browserName !== 'chromium', 'service workers: Chromium only');

const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==',
  'base64',
);

/** Opens the capture page and waits until it works without network. */
async function openCapture(page) {
  await page.goto('/notes/new');
  await page.evaluate(() => navigator.serviceWorker.ready);
  await page.waitForFunction(async () => Boolean(await caches.match('/notes/new')));
}

async function write(page, text) {
  await page.getByLabel('Notiz', { exact: true }).fill(text);
  await page.getByRole('button', { name: 'Speichern' }).click();
}

test('offline: a note with a photo waits and uploads when the network is back', async ({ page, context }) => {
  await signIn(page, { sub: 'otto', name: 'Otto' });
  await openCapture(page);
  await context.setOffline(true);
  await page.reload();

  const text = unique('Offline notiert');
  await page.locator('#pick').setInputFiles({ name: 'zettel.png', mimeType: 'image/png', buffer: PNG });
  await expect(page.locator('#previews img')).toHaveCount(1);
  await write(page, text);
  await expect(page.getByRole('status')).toHaveText('Gespeichert · 1 Notiz wartet auf Netz');
  await expect(page.getByLabel('Notiz', { exact: true })).toHaveValue('');

  await context.setOffline(false);
  await expect(page.getByRole('status')).toHaveText('Gespeichert');
  await page.goto('/notes');
  await page.getByRole('link', { name: text }).click();
  await expect(page.locator('.prose img')).toHaveAttribute('src', /attachments\/zettel\.png$/);
});

test('offline: a lost answer does not create the note twice', async ({ page }) => {
  await signIn(page, { sub: 'otto', name: 'Otto' });
  await openCapture(page);
  let first = true;
  await page.route('**/notes', async (route) => {
    if (route.request().method() === 'POST' && first) {
      first = false;
      await route.fetch(); // the server stores the note …
      await route.abort(); // … but the answer never arrives
      return;
    }
    await route.continue();
  });
  const text = unique('Nur einmal');
  await write(page, text);
  await expect(page.getByRole('status')).toHaveText('Gespeichert · 1 Notiz wartet auf Netz');
  await page.evaluate(() => window.dispatchEvent(new Event('online')));
  await expect(page.getByRole('status')).toHaveText('Gespeichert');
  await page.goto('/notes');
  await expect(page.getByRole('link', { name: text })).toHaveCount(1);
});

test('offline: after the sign-in expired, notes wait until signed in again', async ({ page, context }) => {
  await signIn(page, { sub: 'otto', name: 'Otto' });
  await openCapture(page);
  await context.clearCookies();
  await context.setOffline(true);
  const text = unique('Nach Anmeldung');
  await write(page, text);
  await context.setOffline(false);
  const login = page.getByRole('link', { name: 'Zum Hochladen bitte anmelden' });
  await expect(login).toBeVisible();

  await login.click();
  await page.getByLabel('User id').fill(`otto-${test.info().project.name}`);
  await page.getByLabel('Name').fill('Otto');
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page).toHaveURL(/\/notes\/new$/);
  await expect(page.locator('#outbox')).toBeHidden();
  await page.goto('/notes');
  await expect(page.getByRole('link', { name: text })).toHaveCount(1);
});
