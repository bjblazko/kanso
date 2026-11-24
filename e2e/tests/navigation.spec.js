// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace } from './helpers.js';

const modules = (page) => page.getByRole('navigation', { name: 'Kanso' });
/** A part's link in the bar, whether it is shown there or (on phones) under "Mehr". */
const part = (page, key) => page.locator(`.rail-bar > li[data-key="${key}"] > a`);

/** Opens a part from the bar, or from "Mehr" when it is not in the bar on a phone. */
async function open(page, name) {
  const link = modules(page).getByRole('link', { name, exact: true }).locator('visible=true');
  if (await link.count() === 0) await page.getByText('Mehr', { exact: true }).click();
  await modules(page).getByRole('link', { name, exact: true }).locator('visible=true').click();
}

test('the module bar names each part of Kanso and marks the current one', async ({ page }) => {
  await signIn(page, { sub: 'mona', name: 'Mona' });
  await page.goto('/notes');
  await expect(part(page, 'notes')).toHaveAttribute('aria-current', 'true');
  await expect(part(page, 'wiki')).not.toHaveAttribute('aria-current', 'true');
  await open(page, 'Wiki');
  await expect(page).toHaveURL(/\/wiki$/);
  await expect(part(page, 'wiki')).toHaveAttribute('aria-current', 'true');
  await open(page, 'Start');
  await expect(page).toHaveURL(/\/$/);
  await expect(part(page, 'start')).toHaveAttribute('aria-current', 'true');
  await expect(part(page, 'wiki')).not.toHaveAttribute('aria-current', 'true');
});

test('wiki spaces appear only in the wiki, under their own heading', async ({ page }) => {
  await signIn(page, { sub: 'mona', name: 'Mona', admin: true });
  const space = unique('Modulraum');
  await createSpace(page, space);
  // folded on phones, so look for the link itself rather than its accessible role
  await expect(page.locator('.sidenav a', { hasText: space })).toBeAttached();
  await page.goto('/notes');
  await expect(page.getByRole('navigation', { name: 'Wiki-Bereiche' })).toHaveCount(0);
});

test('group management sits in the user menu, not among the parts', async ({ page }) => {
  await signIn(page, { sub: 'mona', name: 'Mona', admin: true });
  await page.goto('/');
  await expect(modules(page).getByRole('link', { name: 'Gruppen' })).toHaveCount(0);
  await page.locator('.usermenu summary').click();
  await page.getByRole('link', { name: 'Gruppen verwalten' }).click();
  await expect(page).toHaveURL(/\/admin\/groups$/);
});

test('on phones the module bar stays at the bottom, within thumb reach', async ({ page, isMobile }) => {
  test.skip(!isMobile, 'phones only');
  await signIn(page, { sub: 'mona', name: 'Mona' });
  await page.goto('/notes');
  const bar = await modules(page).boundingBox();
  expect(Math.round(bar.y + bar.height)).toBe(page.viewportSize().height);
  const save = await page.goto('/notes/new').then(() => page.getByRole('button', { name: 'Speichern' }).boundingBox());
  expect(save.y + save.height).toBeLessThanOrEqual(bar.y);
});
