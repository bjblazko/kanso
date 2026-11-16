// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, createSpace, createPage } from './helpers.js';

const unique = (name) => `${name} ${test.info().project.name}`;

test('admin creates a space, shares it with a group, member sees it', async ({ browser }) => {
  const project = test.info().project.name;
  const adminCtx = await browser.newContext();
  const memberCtx = await browser.newContext();
  const admin = await adminCtx.newPage();
  const member = await memberCtx.newPage();

  await signIn(member, { sub: 'mira', name: 'Mira' });
  await expect(member.getByText('Noch keine Bereiche.')).toBeVisible();

  await signIn(admin, { sub: 'root', name: 'Root', admin: true });
  await admin.getByRole('link', { name: 'Bereich anlegen' }).click();
  await admin.getByLabel('Name', { exact: true }).fill(unique('Haus'));
  await admin.getByRole('button', { name: 'Bereich anlegen' }).click();
  await expect(admin.getByRole('heading', { level: 1 })).toContainText(unique('Haus'));

  const group = `familie-${project}`;
  await admin.goto('/admin/groups');
  await admin.getByLabel('Name der Gruppe').fill(group);
  await admin.getByRole('button', { name: 'Gruppe anlegen' }).click();
  const section = admin.locator('section.group', { hasText: group });
  await section.getByLabel('Person').selectOption(`mira-${project}`);
  await section.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect(admin.locator('section.group', { hasText: group }).locator('li', { hasText: 'Mira' })).toBeVisible();

  await admin.goto('/');
  await admin.getByRole('link', { name: unique('Haus') }).first().click();
  await admin.getByRole('link', { name: 'Einstellungen' }).click();
  await admin.getByLabel('Art', { exact: true }).selectOption('group');
  await admin.getByLabel('Wer', { exact: true }).selectOption(group);
  await admin.getByLabel('Rolle', { exact: true }).selectOption('editor');
  await admin.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect(admin.locator('.members')).toContainText('Schreiben');

  await member.reload();
  await expect(member.getByRole('link', { name: unique('Haus') }).first()).toBeVisible();
  await member.getByRole('link', { name: unique('Haus') }).first().click();
  await expect(member.getByRole('link', { name: 'Einstellungen' })).toHaveCount(0);

  await adminCtx.close();
  await memberCtx.close();
});

test('non-admins cannot reach group management', async ({ page }) => {
  await signIn(page, { sub: 'nora', name: 'Nora' });
  const response = await page.goto('/admin/groups');
  expect(response.status()).toBe(403);
  await expect(page.getByRole('heading', { name: 'Dafür fehlt dir die Berechtigung.' })).toBeVisible();
});

test('owner deletes a space with its pages', async ({ page }) => {
  const name = unique('Alt');
  await signIn(page, { sub: 'root', name: 'Root', admin: true });
  await createSpace(page, name);
  await createPage(page, 'Notiz');
  await page.getByRole('link', { name, exact: true }).first().click();
  await page.getByRole('link', { name: 'Einstellungen' }).click();
  await page.getByRole('link', { name: 'Bereich löschen' }).click();
  await expect(page.getByText('1 Seiten, davon 0 Rezepte, 0 Anhänge')).toBeVisible();

  const button = page.getByRole('button', { name: `„${name}“ löschen` });
  await button.click();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText(`„${name}“ löschen?`);

  await page.getByLabel('Zum Bestätigen den Namen eintippen').fill(name);
  await button.click();
  await expect(page.getByRole('status')).toHaveText(`Bereich „${name}“ gelöscht.`);
  await expect(page.locator('main').getByRole('link', { name, exact: true })).toHaveCount(0);
});

test('"Was andere sehen" lists a space someone else reads too', async ({ page, browser }) => {
  const readerCtx = await browser.newContext();
  await signIn(await readerCtx.newPage(), { sub: 'ute', name: 'Ute' });
  await readerCtx.close();
  const name = unique('Gemeinsam');
  await signIn(page, { sub: 'olaf', name: 'Olaf', admin: true });
  await createSpace(page, name);
  await page.getByRole('link', { name: 'Einstellungen' }).click();
  await page.getByLabel('Wer').selectOption({ label: 'Ute' });
  await page.getByRole('button', { name: 'Hinzufügen' }).click();

  await page.getByLabel('Menü von Olaf').click();
  await page.getByRole('link', { name: 'Was andere sehen' }).click();
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Was andere sehen');
  const row = page.locator('main li', { has: page.getByRole('link', { name, exact: true }) });
  await expect(row).toContainText('Sichtbar für dich, Ute');
});
