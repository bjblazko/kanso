// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique } from './helpers.js';

/** Fills the capture form and saves; due is the label of a quick choice. */
async function capture(page, title, { urgent = false, important = false, due = null, board = null } = {}) {
  await page.getByLabel('Betreff').fill(title);
  if (board) await page.getByRole('radio', { name: board, exact: true }).check();
  if (urgent) await page.getByLabel('dringend').check();
  if (important) await page.getByLabel('wichtig').check();
  if (due) await page.getByRole('radio', { name: due, exact: true }).check();
  await page.getByRole('button', { name: 'Speichern' }).click();
}

/** The card of a task in a Kanban column (open, doing, waiting, done) or quadrant. */
const card = (page, key, title) => page.locator(`.board-col[data-key="${key}"] .card`, { hasText: title });

test('capture on the phone lands on the board', async ({ page }) => {
  await signIn(page, { sub: 'hanna', name: 'Hanna' });
  await page.goto('/tasks');
  await page.getByRole('link', { name: 'Neue Aufgabe' }).click();
  const title = unique('Hof fegen');
  await capture(page, title, { urgent: true, due: 'heute' });
  await expect(page.getByRole('status')).toHaveText('Gespeichert in Mein Board');
  await expect(page.getByLabel('Betreff')).toHaveValue('');

  await page.getByRole('link', { name: 'Zum Board' }).click();
  const c = card(page, 'open', title);
  await expect(c.locator('.due')).toHaveText('heute');
  await expect(c.locator('abbr')).toHaveText(/^D/);
});

test.describe('offline', () => {
  test.skip(({ browserName }) => browserName !== 'chromium', 'service workers: Chromium only');

  test('a task captured offline is uploaded later', async ({ page, context }) => {
    await signIn(page, { sub: 'olga', name: 'Olga' });
    await page.goto('/tasks/new');
    await page.evaluate(() => navigator.serviceWorker.ready);
    await page.waitForFunction(async () => Boolean(await caches.match('/tasks/new')));
    await context.setOffline(true);
    await page.reload();

    const title = unique('Lohnsteuer machen');
    await capture(page, title, { important: true });
    await expect(page.getByRole('status')).toHaveText('1 Aufgabe wartet auf Netz');

    await context.setOffline(false);
    await expect(page.getByRole('status')).toHaveText('Gespeichert in Mein Board');
    await page.goto('/tasks');
    await expect(card(page, 'open', title)).toBeVisible();
  });
});

test.describe('desktop', () => {
  test.skip(({ isMobile }) => isMobile, 'dragging: desktop only');

  test('drag in Kanban and Matrix shows the same task', async ({ page }) => {
    await signIn(page, { sub: 'dora', name: 'Dora' });
    await page.goto('/tasks/new');
    const title = unique('Lohnsteuer');
    await capture(page, title);
    await expect(page.getByRole('status')).toHaveText('Gespeichert in Mein Board');
    await page.getByRole('link', { name: 'Zum Board' }).click();

    await card(page, 'open', title).dragTo(page.locator('.board-col[data-key="doing"]'));
    await expect(card(page, 'doing', title)).toBeVisible();
    await page.reload();
    await expect(card(page, 'doing', title)).toBeVisible();

    await page.getByRole('link', { name: 'Matrix' }).click();
    await card(page, 'later', title).dragTo(page.locator('.board-col[data-key="do_now"]'));
    await expect(card(page, 'do_now', title)).toBeVisible();
    await page.getByRole('link', { name: 'Kanban' }).click();
    await expect(card(page, 'doing', title).locator('abbr')).toHaveCount(2);
  });
});

test('a family board: capture onto it, another member takes the task', async ({ page }) => {
  const ben = unique('Ben');
  const ida = unique('Ida');
  await signIn(page, { sub: 'benno', name: ben }); // someone to add to the group
  await page.context().clearCookies();
  await signIn(page, { sub: 'ida', name: ida, admin: true });

  const group = unique('Familie');
  await page.goto('/admin/groups');
  await page.getByLabel('Name der Gruppe').fill(group);
  await page.getByRole('button', { name: 'Gruppe anlegen' }).click();
  const section = page.locator('section.group', { has: page.getByRole('heading', { name: group }) });
  for (const person of [ben, ida]) {
    await section.getByLabel('Person').selectOption({ label: person });
    await section.getByRole('button', { name: 'Hinzufügen' }).click();
    await expect(section.locator('.members')).toContainText(person);
  }

  const boardName = unique('Haushalt');
  await page.goto('/admin/boards');
  const create = page.locator('form', { has: page.getByRole('button', { name: 'Board anlegen' }) });
  await create.getByLabel('Name', { exact: true }).fill(boardName);
  await create.getByLabel('Gruppe').selectOption(group);
  await create.getByRole('button', { name: 'Board anlegen' }).click();

  await page.goto('/tasks/new');
  const title = unique('Kind zur Ferienbetreuung anmelden');
  await capture(page, title, { board: boardName, urgent: true, important: true, due: 'morgen' });
  await expect(page.getByRole('status')).toHaveText(`Gespeichert in ${boardName}`);

  await page.context().clearCookies();
  await signIn(page, { sub: 'benno', name: ben });
  await page.goto('/tasks');
  await page.getByRole('link', { name: boardName }).click();
  await card(page, 'open', title).getByRole('link', { name: title }).click();
  await page.getByRole('button', { name: 'Übernehmen' }).click();
  await expect(page.locator('.task-facts')).toContainText('übernommen von dir');

  await page.getByRole('link', { name: 'Bearbeiten' }).click();
  await page.getByRole('radio', { name: 'Wartet' }).check();
  await page.getByLabel('Worauf wartest du?').fill('Antwort Hort');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.getByRole('link', { name: boardName }).click();
  await expect(card(page, 'waiting', title)).toContainText('wartet auf: Antwort Hort');
});
