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

/** Makes a name unique per browser project (projects share one server). */
export const unique = (name) => `${name} ${test.info().project.name}`;

/** Creates a space as the signed-in admin and opens it. */
export async function createSpace(page, name) {
  await page.goto('/admin/spaces/new');
  await page.getByLabel('Name', { exact: true }).fill(name);
  await page.getByRole('button', { name: 'Bereich anlegen' }).click();
}

/** Creates a page in the open space (or below the open page). */
export async function createPage(page, title, { child = false } = {}) {
  await page.getByRole('link', { name: child ? 'Unterseite anlegen' : 'Seite anlegen' }).click();
  await page.getByLabel('Titel').fill(title);
  await page.getByRole('button', { name: 'Seite anlegen' }).click();
}
