// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

const all = ['start', 'notes', 'todos', 'tasks', 'wiki', 'kitchen', 'focus'];

/** The keys of the rail's parts in the bar that are not hidden, in order. */
const barKeys = (page) => page.locator('.rail-bar > li[data-key]:not([hidden])')
  .evaluateAll((lis) => lis.map((li) => li.dataset.key));

/** The parts a person sees in the bar right now (on phones without those under "Mehr"). */
const shownKeys = (page) => page.locator('.rail-bar > li[data-key]:visible')
  .evaluateAll((lis) => lis.map((li) => li.dataset.key));

async function setMenu(page, value) {
  await page.evaluate((v) => {
    localStorage.setItem(`kanso.menu.${document.querySelector('.rail').dataset.user}`, v);
  }, value);
  await page.reload();
}

test('hide and reorder parts; the choice stays on this device', async ({ page }) => {
  await signIn(page, { sub: 'mia', name: 'Mia' });
  await page.goto('/me/menu');
  await page.getByRole('checkbox', { name: 'Wiki' }).uncheck();
  await page.getByRole('button', { name: 'Fokus nach oben' }).click();
  await page.getByRole('button', { name: 'Fokus nach oben' }).click();
  await expect(page.getByRole('checkbox', { name: 'Start' })).toBeDisabled();

  await page.goto('/notes');
  expect(await barKeys(page)).toEqual(['start', 'notes', 'todos', 'tasks', 'focus', 'kitchen']);
  await page.goto('/wiki');
  await expect(page.locator('h1')).toBeVisible(); // hidden, still reachable

  await page.goto('/me/menu');
  await page.getByRole('button', { name: 'Zurücksetzen' }).click();
  expect(await barKeys(page)).toEqual(all);
});

test('broken settings fall back to the default menu', async ({ page }) => {
  await signIn(page, { sub: 'mia', name: 'Mia' });
  await page.goto('/notes');
  await setMenu(page, '{broken');
  expect(await barKeys(page)).toEqual(all);
  await setMenu(page, '{"order":["focus","nope"],"hidden":["start","wiki","nope"]}');
  expect(await barKeys(page)).toEqual(['focus', 'start', 'notes', 'todos', 'tasks', 'kitchen']);
});

test.describe('phone', () => {
  test.skip(({ isMobile }) => !isMobile, 'the bottom bar: phones only');

  test('four parts and Mehr; Mehr holds the rest and the settings', async ({ page }) => {
    await signIn(page, { sub: 'mia', name: 'Mia' });
    await page.goto('/notes');
    await setMenu(page, '{}');
    expect(await shownKeys(page)).toEqual(['start', 'notes', 'todos', 'tasks']);
    await page.getByText('Mehr', { exact: true }).click();
    const list = page.locator('.rail-more-list');
    await expect(list.getByRole('link')).toHaveText(['Wiki', 'Küche', 'Fokus', 'Menü anpassen']);
    await list.getByRole('link', { name: 'Fokus' }).click();
    await expect(page).toHaveURL(/\/focus$/);
    await expect(page.locator('.rail-more')).toHaveClass(/rail-more-current/);
  });

  test('five parts or fewer need no Mehr', async ({ page }) => {
    await signIn(page, { sub: 'mia', name: 'Mia' });
    await page.goto('/notes');
    await setMenu(page, '{"hidden":["notes","todos","tasks","wiki","kitchen","focus"]}');
    expect(await shownKeys(page)).toEqual(['start']);
    await expect(page.locator('.rail-more')).toBeHidden();
  });
});
