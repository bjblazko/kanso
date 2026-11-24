// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

async function at(page) {
  await page.clock.install({ time: new Date('2026-11-24T08:00:00') });
  await page.clock.pauseAt(new Date('2026-11-24T08:00:01'));
  await signIn(page, { sub: 'nina', name: 'Nina' });
  await page.goto('/care');
  await page.evaluate(() => localStorage.clear());
  await page.reload();
}

const bar = (page) => page.locator('#nudge');

test('after the chosen time a quiet bar suggests a step in between', async ({ page }) => {
  await at(page);
  await page.getByLabel('Einen Übergang vorschlagen nach').selectOption('30');
  await page.clock.fastForward('29:00');
  await expect(bar(page)).toBeHidden();
  await page.clock.fastForward('01:30');
  await expect(bar(page)).toBeVisible();
  await bar(page).getByRole('button', { name: 'Später' }).click();
  await expect(bar(page)).toBeHidden();
  await page.clock.fastForward('15:30');
  await expect(bar(page)).toBeVisible();
  await bar(page).getByRole('button', { name: 'Schließen' }).click();
  await expect(bar(page)).toBeHidden();
  await page.clock.fastForward('20:00');
  await expect(bar(page)).toBeHidden();
});

test('no reminder while a Fokus round runs, none when switched off', async ({ page }) => {
  await at(page);
  await page.getByLabel('Einen Übergang vorschlagen nach').selectOption('30');
  await page.evaluate(() => {
    const user = document.querySelector('.rail').dataset.user;
    localStorage.setItem(`kanso.focus.${user}`, JSON.stringify({ endsAt: Date.now() + 3 * 3600_000 }));
  });
  await page.clock.fastForward('45:00');
  await expect(bar(page)).toBeHidden();
  await page.evaluate(() => localStorage.removeItem(`kanso.focus.${document.querySelector('.rail').dataset.user}`));
  await page.getByLabel('Einen Übergang vorschlagen nach').selectOption('0');
  await page.clock.fastForward('03:00:00');
  await expect(bar(page)).toBeHidden();
});

test('in between: three steps, another idea keeps what was typed, a note on request', async ({ page }) => {
  await signIn(page, { sub: 'nina', name: 'Nina' });
  await page.goto('/care');
  await page.getByRole('link', { name: /Zwischendurch/ }).first().click();
  await expect(page).toHaveURL(/\/care\/between$/);
  await page.getByLabel('Was hast du gerade geschafft?').fill('Steuer fertig');
  const first = await page.locator('#between-tip-title').textContent();
  await page.getByRole('button', { name: 'Andere Idee' }).click();
  await expect(page.locator('#between-tip-title')).not.toHaveText(first);
  await expect(page.getByLabel('Was hast du gerade geschafft?')).toHaveValue('Steuer fertig');
  await page.getByRole('button', { name: 'Als Notiz' }).click();
  await expect(page.locator('main')).toContainText('Steuer fertig');
});
