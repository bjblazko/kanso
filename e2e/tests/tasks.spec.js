// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique } from './helpers.js';

/** Fills the capture form and saves; due is the label of a quick choice. */
async function capture(page, title, { urgent = false, important = false, due = null, board = null } = {}) {
  await page.getByLabel('Betreff').fill(title);
  if (board) await page.getByRole('radio', { name: board, exact: true }).check();
  if (urgent) await page.getByLabel('dringend').check();
  if (important) await page.getByLabel('wichtig').check();
  if (due) await page.getByRole('radio', { name: due, exact: true }).check();
  await page.getByRole('button', { name: 'Speichern' }).click();
}

/** The card of a task in a Kanban column (open, doing, waiting, done) or quadrant. */
const card = (page, key, title) => page.locator(`.board-col[data-key="${key}"] .card`, { hasText: title });

test('capture on the phone lands on the board', async ({ page }) => {
  await signIn(page, { sub: 'hanna', name: 'Hanna' });
  await page.goto('/tasks');
  await page.getByRole('link', { name: 'Neue Aufgabe' }).click();
  const title = unique('Hof fegen');
  await capture(page, title, { urgent: true, due: 'heute' });
  await expect(page.getByRole('status')).toHaveText('Gespeichert in Mein Board');
  await expect(page.getByLabel('Betreff')).toHaveValue('');

  await page.getByRole('link', { name: 'Zum Board' }).click();
  const c = card(page, 'open', title);
  await expect(c.locator('.due')).toHaveText('heute');
  await expect(c.locator('abbr')).toHaveText(/^D/);
});

test.describe('offline', () => {
  test.skip(({ browserName }) => browserName !== 'chromium', 'service workers: Chromium only');

  test('a task captured offline is uploaded later', async ({ page, context }) => {
    await signIn(page, { sub: 'olga', name: 'Olga' });
    await page.goto('/tasks/new');
    await page.evaluate(() => navigator.serviceWorker.ready);
    await page.waitForFunction(async () => Boolean(await caches.match('/tasks/new')));
    await context.setOffline(true);
    await page.reload();

    const title = unique('Lohnsteuer machen');
    await capture(page, title, { important: true });
    await expect(page.getByRole('status')).toHaveText('1 Aufgabe wartet auf Netz');

    await context.setOffline(false);
    await expect(page.getByRole('status')).toHaveText('Gespeichert in Mein Board');
    await page.goto('/tasks');
    await expect(card(page, 'open', title)).toBeVisible();
  });
});
