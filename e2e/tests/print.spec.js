// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

test('print shows the content only, and the button opens the print dialog', async ({ page }) => {
  await signIn(page, { sub: 'pia', name: 'Pia' });
  const title = `Zum Ausdrucken ${Date.now()}`;
  await page.goto('/notes/new');
  await page.getByLabel('Notiz', { exact: true }).fill(`${title}\nZeile zwei`);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page.getByRole('status')).toHaveText('Gespeichert');
  await page.goto('/notes');
  await page.getByRole('link', { name: title }).click();
  await page.evaluate(() => { window.__printed = 0; window.print = () => { window.__printed += 1; }; });
  await page.getByRole('button', { name: 'Drucken' }).click();
  expect(await page.evaluate(() => window.__printed)).toBe(1);

  await page.emulateMedia({ media: 'print' });
  await expect(page.locator('h1')).toBeVisible();
  for (const hidden of ['.topbar', '.rail', '.page-toolbar', '.footer']) {
    await expect(page.locator(hidden)).toBeHidden();
  }
});
