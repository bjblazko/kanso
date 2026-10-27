// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique } from './helpers.js';

/** Creates a list from the overview and waits until it is open. */
export async function createList(page, name) {
  await page.goto('/todos');
  await page.getByLabel('Neue Liste').fill(name);
  await page.getByRole('button', { name: 'Anlegen' }).click();
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();
}

/** Adds an item at the end of the open list. */
export async function addItem(page, text) {
  await page.getByLabel('Neuer Punkt').fill(text);
  await page.getByLabel('Neuer Punkt').press('Enter');
  await expect(page.getByRole('checkbox', { name: text })).toBeVisible();
}

/** Waits until the list shows exactly these items, in this order. */
const expectItems = (page, items) => expect(page.locator('.todo-item .todo-text')).toHaveText(items);

test('a list with items and a sub-item', async ({ page }) => {
  await signIn(page, { sub: 'tilda', name: 'Tilda' });
  await createList(page, unique('Gartenarbeit'));
  await addItem(page, 'Laub wegfegen');
  await addItem(page, 'Rasen mähen');

  await page.getByRole('button', { name: 'Laub wegfegen' }).click();
  await page.getByRole('button', { name: 'Unterpunkt hinzufügen' }).click();
  await page.getByLabel('Punkt', { exact: true }).fill('vor der Haustür');
  await page.getByLabel('Punkt', { exact: true }).press('Enter');
  await page.getByLabel('Punkt', { exact: true }).fill('Garten');
  await page.getByLabel('Punkt', { exact: true }).press('Enter');
  await page.getByLabel('Punkt', { exact: true }).press('Escape');

  await expectItems(page, ['Laub wegfegen', 'vor der Haustür', 'Garten', 'Rasen mähen']);
  // measured in one go: a sync may redraw the list between two separate lookups
  const indent = await page.evaluate(() => {
    const x = (name) => document.querySelector(`.todo-check input[aria-label="${name}"]`).getBoundingClientRect().x;
    return x('Garten') - x('Laub wegfegen');
  });
  expect(indent).toBeGreaterThan(16);

  await page.reload(); // the server has it all
  await expectItems(page, ['Laub wegfegen', 'vor der Haustür', 'Garten', 'Rasen mähen']);
});

test('ticking keeps the order and fades the item', async ({ page }) => {
  await signIn(page, { sub: 'tilda', name: 'Tilda' });
  await createList(page, unique('Weihnachten'));
  for (const t of ['Baum kaufen', 'Plätzchen backen', 'Geschenke']) await addItem(page, t);

  await page.getByRole('checkbox', { name: 'Baum kaufen' }).check();
  await expectItems(page, ['Baum kaufen', 'Plätzchen backen', 'Geschenke']);
  const done = page.locator('.todo-item', { hasText: 'Baum kaufen' });
  await expect(done).toHaveCSS('opacity', '0.62');
  await expect(page.locator('.todo-item', { hasText: 'Geschenke' })).toHaveCSS('opacity', '1');

  await page.reload();
  await expect(page.getByRole('checkbox', { name: 'Baum kaufen' })).toBeChecked();
});

test('an item can be renamed, indented and deleted', async ({ page }) => {
  await signIn(page, { sub: 'tilda', name: 'Tilda' });
  await createList(page, unique('Umbau'));
  await addItem(page, 'Wand');
  await addItem(page, 'Fenstr');
  await page.getByRole('button', { name: 'Fenstr' }).click();
  await page.getByLabel('Punkt', { exact: true }).fill('Fenster');
  await page.getByRole('button', { name: 'Einrücken' }).click();
  await expect(page.locator('.todo-item.sub .todo-text')).toHaveText('Fenster');
  await page.getByRole('button', { name: 'Wand' }).click();
  await page.getByRole('button', { name: 'Punkt löschen' }).click();
  await expectItems(page, []); // the main item takes its sub-item along
});

test('the overview shows the most recently changed list first', async ({ page }) => {
  await signIn(page, { sub: 'udo', name: 'Udo' });
  const older = unique('Ältere');
  const newer = unique('Neuere');
  await createList(page, older);
  await addItem(page, 'Eins');
  await createList(page, newer);
  await page.getByRole('link', { name: 'Alle Listen' }).click();
  await expect(page.locator('.todo-lists a').first()).toHaveText(newer);

  await page.getByRole('link', { name: older }).click();
  await page.getByRole('checkbox', { name: 'Eins' }).check();
  await page.getByRole('link', { name: 'Alle Listen' }).click();
  await expect(page.locator('.todo-lists a').first()).toHaveText(older);
  await expect(page.locator('.todo-lists li').first()).toContainText('1 von 1 erledigt');
});

test('a shared list: the other person ticks, only the owner deletes', async ({ browser }) => {
  const annaCtx = await browser.newContext({ locale: 'de-DE' });
  const benCtx = await browser.newContext({ locale: 'de-DE' });
  const anna = await annaCtx.newPage();
  const ben = await benCtx.newPage();
  const benName = unique('Ben'); // both browser projects share one server
  await signIn(ben, { sub: 'bert', name: benName }); // not 'ben': shell.spec switches him to English
  await signIn(anna, { sub: 'anna', name: 'Anna' });
  const name = unique('Einkauf Fest');
  await createList(anna, name);
  await addItem(anna, 'Kerzen');
  await anna.getByRole('link', { name: 'Teilen …' }).click();
  await anna.getByLabel('Wer').selectOption({ label: benName });
  await anna.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect(anna.locator('.members')).toContainText(benName);

  await ben.goto('/todos');
  await expect(ben.locator('.todo-lists li', { hasText: name })).toContainText('von Anna');
  await ben.getByRole('link', { name }).click();
  await Promise.all([ben.waitForResponse('**/api/todos/sync'), ben.getByRole('checkbox', { name: 'Kerzen' }).check()]);
  await expect(ben.getByRole('link', { name: 'Liste löschen …' })).toHaveCount(0);
  await expect(ben.getByRole('status')).toHaveText('');

  await anna.goto('/todos');
  await anna.getByRole('link', { name }).click();
  await expect(anna.getByRole('checkbox', { name: 'Kerzen' })).toBeChecked();
  await anna.getByRole('link', { name: 'Liste löschen …' }).click();
  await anna.getByRole('button', { name: `„${name}“ löschen` }).click();
  await expect(anna).toHaveURL(/\/todos$/);
  await expect(anna.getByRole('link', { name })).toHaveCount(0);
  await annaCtx.close();
  await benCtx.close();
});

test('while an item is edited, a tap on a link or a checkbox still works', async ({ page }) => {
  await signIn(page, { sub: 'tilda', name: 'Tilda' });
  await createList(page, unique('Fokus'));
  await addItem(page, 'Eins');
  await addItem(page, 'Zwei');
  await page.getByRole('button', { name: 'Eins' }).click();
  await page.getByRole('checkbox', { name: 'Zwei' }).check();
  await expect(page.getByRole('checkbox', { name: 'Zwei' })).toBeChecked();
  await page.getByRole('button', { name: 'Eins' }).click();
  await page.getByRole('link', { name: 'Alle Listen' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Todos' })).toBeVisible();
});
