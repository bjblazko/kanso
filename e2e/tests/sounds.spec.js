// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

async function open(page) {
  await page.clock.install({ time: new Date('2026-11-24T22:00:00') });
  await page.clock.pauseAt(new Date('2026-11-24T22:00:01'));
  await signIn(page, { sub: 'sven', name: 'Sven' });
  await page.goto('/care/sounds');
  await page.evaluate(() => localStorage.clear());
  await page.reload();
}

test('sounds play with a sleep timer and stop by themselves', async ({ page }) => {
  const errors = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await open(page);
  await page.getByLabel('Schlaf-Timer').selectOption('15');
  await page.getByLabel('Wellen').check();
  await page.getByRole('button', { name: 'Abspielen' }).click();
  await expect(page.locator('#sounds-toggle')).toHaveAttribute('data-running', 'true');
  await expect(page.locator('#sounds-time')).toHaveText('15:00');
  await page.clock.fastForward('05:00');
  await expect(page.locator('#sounds-time')).toHaveText('10:00');
  await expect(page.locator('#sounds-meter')).toHaveAttribute('aria-valuenow', '33');
  const busy = await page.evaluate(() => localStorage.getItem(`kanso.busy.${document.querySelector('.rail').dataset.user}`));
  expect(Number(busy)).toBeGreaterThan(0); // the reminder keeps quiet
  await page.clock.fastForward('10:01');
  await expect(page.locator('#sounds-toggle')).toHaveAttribute('data-running', 'false');
  await expect(page.locator('#sounds-time')).toHaveText('15:00');
  expect(errors).toEqual([]);
});

test('endless, and the choices stay on the device', async ({ page }) => {
  await open(page);
  await page.getByLabel('Schlaf-Timer').selectOption('0');
  await page.getByLabel('Braun (tief)').check();
  await page.getByLabel('Ruhiger Akkord mit leisen Glocken').uncheck();
  await expect(page.locator('#sounds-time')).toHaveText('endlos');
  await expect(page.locator('#sounds-meter')).toBeHidden();
  await page.reload();
  await expect(page.getByLabel('Braun (tief)')).toBeChecked();
  await expect(page.getByLabel('Ruhiger Akkord mit leisen Glocken')).not.toBeChecked();
  await page.getByRole('button', { name: 'Abspielen' }).click();
  await page.clock.fastForward('02:00:00');
  await expect(page.locator('#sounds-toggle')).toHaveAttribute('data-running', 'true');
  await page.getByRole('button', { name: 'Beenden' }).click();
  await expect(page.locator('#sounds-toggle')).toHaveAttribute('data-running', 'false');
});
