// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique } from './helpers.js';

test('notes: write one, find it in the list, delete it', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  const text = unique('Zählerstand Keller');
  await page.goto('/notes/new');
  await page.getByLabel('Notiz', { exact: true }).fill(`${text}\n4711 kWh`);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Gespeichert');
  await expect(page.getByLabel('Notiz', { exact: true })).toHaveValue('');

  await page.getByRole('link', { name: 'Alle Notizen' }).click();
  await page.getByRole('link', { name: text }).click();
  await expect(page.getByRole('heading', { level: 1, name: text })).toBeVisible();
  await expect(page.locator('.prose')).toContainText('4711 kWh');

  await page.getByRole('link', { name: 'Notiz löschen …' }).click();
  await page.getByRole('button', { name: `„${text}“ löschen` }).click();
  await expect(page).toHaveURL(/\/notes$/);
  await expect(page.getByRole('link', { name: text })).toHaveCount(0);
});

test('notes: an empty note is refused with a reason', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  await page.goto('/notes/new');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('alert')).toHaveText('Schreib etwas oder hänge ein Foto an.');
});
