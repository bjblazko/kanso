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

test('meditation plan: the fill pause makes up the minutes', async ({ page }) => {
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care');
  const p = await page.evaluate(async () => (await import('/static/js/meditation-plan.js')).plan([10, 20, 5], [4, 6, 2], 1, 1, 6));
  expect(p).toEqual({ starts: [6, 20, 47], endBell: 54, total: 60 });
});

/** A short silent WAV, so the player has something to decode without the real clips. */
function silentWav(seconds = 0.5, rate = 8000) {
  const samples = Math.round(seconds * rate);
  const b = Buffer.alloc(44 + samples * 2);
  b.write('RIFF', 0); b.writeUInt32LE(36 + samples * 2, 4); b.write('WAVE', 8); b.write('fmt ', 12);
  b.writeUInt32LE(16, 16); b.writeUInt16LE(1, 20); b.writeUInt16LE(1, 22); b.writeUInt32LE(rate, 24);
  b.writeUInt32LE(rate * 2, 28); b.writeUInt16LE(2, 32); b.writeUInt16LE(16, 34); b.write('data', 36); b.writeUInt32LE(samples * 2, 40);
  return b;
}

test('a meditation loads the chosen voice, counts down and can be ended', async ({ page }) => {
  const asked = [];
  await page.route('**/static/audio/meditation/**', (route) => {
    asked.push(new URL(route.request().url()).pathname);
    return route.fulfill({ contentType: 'audio/wav', body: silentWav() });
  });
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care/m/arrive');
  await page.getByLabel(/Orion/).check();
  await page.getByRole('button', { name: 'Starten' }).click();
  await expect(page.locator('#med-time')).toHaveText(/^0[45]:[0-5]\d$/);
  await expect(page.getByLabel(/Lyra/)).toBeDisabled();
  expect(asked.length).toBe(13);
  expect(asked.every((p) => p.startsWith('/static/audio/meditation/orion/de/'))).toBe(true);
  await expect(page.locator('#med-now')).toHaveText(/^Willkommen/, { timeout: 10_000 });
  await page.getByRole('button', { name: 'Beenden' }).click();
  await expect(page.locator('#med-time')).toHaveText('05:00');
  await expect(page.getByLabel(/Lyra/)).toBeEnabled();
});
