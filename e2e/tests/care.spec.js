// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

test('three ways in; "zu viel" puts help first and offers another calming step', async ({ page }) => {
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care');
  await expect(page.locator('.care-ways a')).toHaveCount(3);
  await page.getByRole('link', { name: /Es ist gerade zu viel/ }).click();
  await expect(page).toHaveURL(/\/care\/sos$/);
  await expect(page.locator('.care > :nth-child(2)')).toHaveClass(/care-help/);
  await expect(page.getByRole('link', { name: '0800 111 0 111' })).toHaveAttribute('href', 'tel:08001110111');
  const first = await page.locator('#care-now').textContent();
  await page.getByRole('link', { name: 'Andere Idee' }).click();
  await expect(page.locator('#care-now')).not.toHaveText(first);
});

test('breathing counts along and stops after six rounds', async ({ page }) => {
  await page.clock.install({ time: new Date('2026-11-24T08:00:00') });
  await page.clock.pauseAt(new Date('2026-11-24T08:00:01'));
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care/breathe');
  await page.getByRole('button', { name: 'Mitzählen' }).click();
  await expect(page.locator('#breath')).toHaveText('Einatmen 1 · Runde 1 von 6');
  await page.clock.runFor(3000);
  await expect(page.locator('#breath-meter .time-meter-fill')).toHaveAttribute('style', /width: 100%/);
  await page.clock.runFor(1000);
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

test('meditation plan: the time left over is shared among the pauses', async ({ page }) => {
  await signIn(page, { sub: 'carl', name: 'Carl' });
  await page.goto('/care');
  const p = await page.evaluate(async () => (await import('/static/js/meditation-plan.js')).plan([10, 20, 5], [4, 6, 2], 1, 6));
  // 59 s of their own, 1 s left over, shared 4:6:2 among the pauses
  expect(p.starts[0]).toBe(6);
  expect(p.starts[1]).toBeCloseTo(20 + 1 / 3, 6);
  expect(p.starts[2]).toBeCloseTo(46 + 5 / 6, 6);
  expect(p.total).toBeCloseTo(60, 6);
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
  await expect(page.locator('#med-meter')).toHaveAttribute('data-state', 'running');
  await expect(page.locator('#med-toggle .when-running')).toBeVisible(); // a stop icon, not play
  await expect(page.locator('#med-toggle .when-idle')).toBeHidden();
  await page.getByRole('button', { name: 'Beenden' }).click();
  await expect(page.locator('#med-time')).toHaveText('05:00');
  await expect(page.locator('#med-meter')).toHaveAttribute('aria-valuenow', '0');
  await expect(page.getByLabel(/Lyra/)).toBeEnabled();
});

test('own ideas: add, see in first aid, delete', async ({ page }) => {
  await signIn(page, { sub: 'ida', name: 'Ida' });
  await page.goto('/care/settings');
  const idea = `Katze streicheln ${Date.now()}`;
  await page.getByLabel('Idee', { exact: true }).fill(idea);
  await page.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect(page.locator('.care-own-row strong')).toContainText([idea]);
  await page.getByRole('button', { name: `„${idea}“ löschen` }).click();
  await expect(page.locator('.care-own-row')).toHaveCount(0);
});
