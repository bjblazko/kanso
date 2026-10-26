// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

const unique = (name) => `${name} ${test.info().project.name}`;

test('admin creates a space, shares it with a group, member sees it', async ({ browser }) => {
  const project = test.info().project.name;
  const adminCtx = await browser.newContext();
  const memberCtx = await browser.newContext();
  const admin = await adminCtx.newPage();
  const member = await memberCtx.newPage();

  await signIn(member, { sub: 'mia', name: 'Mia' });
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
  await section.getByLabel('Person').selectOption(`mia-${project}`);
  await section.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect(admin.locator('section.group', { hasText: group }).locator('li', { hasText: 'Mia' })).toBeVisible();

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
