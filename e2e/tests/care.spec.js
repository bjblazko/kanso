// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

test('first aid offers another idea; an empty battery puts help first', async ({ page }) => {
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care');
  const first = await page.locator('#care-now').textContent();
  await page.getByRole('link', { name: 'Andere Idee' }).click();
  await expect(page.locator('#care-now')).not.toHaveText(first);
  await page.getByRole('link', { name: /Leer/ }).click();
  await expect(page.locator('.care > :first-child')).toHaveClass(/care-help/);
  await expect(page.getByRole('link', { name: '0800 111 0 111' })).toHaveAttribute('href', 'tel:08001110111');
});

test('breathing counts along and stops after six rounds', async ({ page }) => {
  await page.clock.install({ time: new Date('2026-11-24T08:00:00') });
  await page.clock.pauseAt(new Date('2026-11-24T08:00:01'));
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care/breathe');
  await page.getByRole('button', { name: 'Mitzählen' }).click();
  await expect(page.locator('#breath')).toHaveText('Einatmen 1 · Runde 1 von 6');
  await page.clock.runFor(4000);
  await expect(page.locator('#breath')).toHaveText('Ausatmen 1 · Runde 1 von 6');
  await page.clock.runFor(60_000);
  await expect(page.locator('#breath')).toHaveText(/^Fertig/);
  await expect(page.getByRole('button', { name: 'Mitzählen' })).toBeVisible();
});

test('a check-in becomes a private note on request', async ({ page }) => {
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care/checkin');
  await page.getByLabel(/Was kostet dich/).fill('Der Umzug');
  await page.getByRole('button', { name: 'Als Notiz speichern' }).click();
  await expect(page).toHaveURL(/\/notes\/[a-z2-7]+$/);
  await expect(page.locator('main')).toContainText('Der Umzug');
});
