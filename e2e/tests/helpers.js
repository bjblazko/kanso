// SPDX-License-Identifier: Apache-2.0
import { test } from '@playwright/test';

/**
 * Signs in through the dev login form. The user id is made unique per
 * browser project, because all projects share one server and its data.
 */
export async function signIn(page, { sub = 'anna', name = 'Anna', admin = false } = {}) {
  await page.goto('/');
  await page.getByLabel('User id').fill(`${sub}-${test.info().project.name}`);
  await page.getByLabel('Name').fill(name);
  if (admin) await page.getByLabel('Admin').check();
  await page.getByRole('button', { name: 'Sign in' }).click();
}
