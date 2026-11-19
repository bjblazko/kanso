// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, useAsCookbook } from './helpers.js';

const item = (page, text) => page.locator('.todo-item', { has: page.getByRole('checkbox', { name: text }) });

test('shopping list: amounts with − and + in sensible steps, a note, filled from a recipe', async ({ page }) => {
  await signIn(page, { sub: 'sina', name: 'Sina', admin: true });
  const name = unique('Wocheneinkauf');
  await page.goto('/todos');
  await page.getByLabel('Neue Liste').fill(name);
  await page.getByRole('button', { name: 'Einkaufsliste anlegen' }).click();
  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();

  const add = async (text) => {
    await page.getByLabel('Was fehlt?').fill(text);
    await page.getByLabel('Was fehlt?').press('Enter');
    await expect(page.getByRole('checkbox', { name: text })).toBeVisible();
  };
  const step = async (text, which) => item(page, text).getByRole('button', { name: which }).click();
  const shows = (text) => expect(page.getByRole('checkbox', { name: text })).toBeVisible();

  await add('2 Zitronen');
  await step('2 Zitronen', 'Eins mehr');
  await shows('3 Zitronen');
  await step('3 Zitronen', 'Eins weniger');
  await step('2 Zitronen', 'Eins weniger');
  await expect(item(page, '1 Zitronen').getByRole('button', { name: 'Eins weniger' })).toBeDisabled();

  await add('250 g Butter');
  await step('250 g Butter', 'Eins mehr');
  await shows('300 g Butter');
  await add('87 g Parmesan');
  await step('87 g Parmesan', 'Eins mehr');
  await shows('90 g Parmesan');
  await step('90 g Parmesan', 'Eins weniger');
  await shows('80 g Parmesan');
  await step('300 g Butter', 'Menge ändern');
  await page.getByLabel('Menge', { exact: true }).fill('275');
  await page.getByLabel('Menge', { exact: true }).press('Enter');
  await shows('275 g Butter');
  await add('1,5 kg Mehl');
  await step('1,5 kg Mehl', 'Eins mehr');
  await shows('2 kg Mehl');

  await add('1 l Milch');
  await item(page, '1 l Milch').getByRole('button', { name: 'Milch', exact: true }).click();
  await page.getByLabel('Notiz').fill('laktosefrei, sonst Hafermilch');
  await page.getByLabel('Notiz').press('Enter');
  await expect(item(page, '1 l Milch')).toContainText('laktosefrei, sonst Hafermilch');
  await expect(page.locator('#todos-status')).toHaveText(''); // synced, nothing waiting
  await page.reload();
  await expect(item(page, '1 l Milch')).toContainText('laktosefrei, sonst Hafermilch');

  const cookbook = unique('Rezepte Einkauf');
  await createSpace(page, cookbook);
  await useAsCookbook(page, cookbook);
  await page.goto('/recipes/new');
  const recipe = unique('Milchreis');
  await page.getByLabel('Titel').fill(recipe);
  await page.getByRole('button', { name: 'Rezept anlegen' }).click();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill('## Zutaten\n\n- 1 l Milch\n- 250 g Milchreis\n');
  await page.getByRole('button', { name: 'Speichern' }).click();
  await page.getByRole('link', { name: 'Einkaufen' }).click();
  await expect(page.getByRole('button', { name: 'Vorschau aktualisieren' })).toBeHidden(); // updates by itself
  await page.getByLabel('Auf welche Einkaufsliste?').selectOption({ label: name });
  await expect(page.locator('.shop-lines')).toContainText('1 l Milch → 2 l Milch');
  await page.getByRole('button', { name: 'Auf die Liste' }).click();

  await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();
  await expect(item(page, '2 l Milch')).toContainText(`für ${recipe}`);
  await expect(item(page, '250 g Milchreis')).toContainText(`für ${recipe}`);
});

test('shopping list: an item not got moves to another list and adds up there', async ({ page }) => {
  await signIn(page, { sub: 'sina', name: 'Sina', admin: true });
  const newList = async (name) => {
    await page.goto('/todos');
    await page.getByLabel('Neue Liste').fill(name);
    await page.getByRole('button', { name: 'Einkaufsliste anlegen' }).click();
    await expect(page.getByRole('heading', { level: 1, name })).toBeVisible();
  };
  const add = async (text) => {
    await page.getByLabel('Was fehlt?').fill(text);
    await page.getByLabel('Was fehlt?').press('Enter');
    await expect(page.getByRole('checkbox', { name: text })).toBeVisible();
  };
  const market = unique('Wochenmarkt');
  await newList(market);
  await add('300 g Mehl');
  const week = unique('KW 46');
  await newList(week);
  await add('200 g Mehl');
  await add('1 Brokkoli');

  await item(page, '200 g Mehl').getByRole('button', { name: 'Mehl', exact: true }).click();
  await page.getByLabel('Auf andere Liste').selectOption({ label: market });
  await page.getByRole('button', { name: 'Verschieben' }).click();
  await expect(page.getByRole('checkbox', { name: '200 g Mehl' })).toHaveCount(0);

  await item(page, '1 Brokkoli').getByRole('button', { name: 'Brokkoli', exact: true }).click();
  await page.getByLabel('Auf andere Liste').selectOption({ index: (await page.locator('#shop-move option').count()) - 1 }); // the new list
  await page.getByRole('button', { name: 'Verschieben' }).click();
  await expect(page.getByRole('checkbox', { name: '1 Brokkoli' })).toHaveCount(0);

  await page.goto('/todos');
  await page.getByRole('link', { name: market }).click();
  await expect(page.getByRole('checkbox', { name: '500 g Mehl' })).toBeVisible(); // added up by the server
  await page.goto('/todos');
  await page.getByRole('link', { name: /^Einkauf \d/ }).first().click();
  await expect(page.getByRole('checkbox', { name: '1 Brokkoli' })).toBeVisible();
});

test('shopping list: the amount sits at − and +, also for "3x", and an item without one counts as 1', async ({ page }) => {
  await signIn(page, { sub: 'sina', name: 'Sina', admin: true });
  const name = unique('Getränke');
  await page.goto('/todos');
  await page.getByLabel('Neue Liste').fill(name);
  await page.getByRole('button', { name: 'Einkaufsliste anlegen' }).click();
  const add = async (text) => {
    await page.getByLabel('Was fehlt?').fill(text);
    await page.getByLabel('Was fehlt?').press('Enter');
    await expect(page.getByRole('checkbox', { name: text })).toBeVisible();
  };
  const shows = async (text, label, amount) => {
    await expect(item(page, text).locator('.todo-text')).toHaveText(label);
    await expect(item(page, text).locator('.shop-amount')).toHaveText(amount);
  };
  const more = (text) => item(page, text).getByRole('button', { name: 'Eins mehr' }).click();

  await add('Bier Alkoholfrei');
  await shows('Bier Alkoholfrei', 'Bier Alkoholfrei', '1');
  await expect(item(page, 'Bier Alkoholfrei').getByRole('button', { name: 'Eins weniger' })).toBeDisabled();
  await more('Bier Alkoholfrei');
  await shows('2 Bier Alkoholfrei', 'Bier Alkoholfrei', '2');

  await add('3x Wasser');
  await shows('3x Wasser', 'Wasser', '3');
  await more('3x Wasser');
  await shows('4x Wasser', 'Wasser', '4');
  await add('3 x Saft');
  await more('3 x Saft');
  await shows('4 x Saft', 'Saft', '4');

  await add('250 g Butter');
  await shows('250 g Butter', 'Butter', '250 g');

  await add('7Up');
  await expect(item(page, '7Up').locator('.todo-text')).toHaveText('7Up');
  await expect(item(page, '7Up').getByRole('button', { name: 'Eins mehr' })).toHaveCount(0);
  await add('2 - 3 Eier');
  await expect(item(page, '2 - 3 Eier').getByRole('button', { name: 'Eins mehr' })).toHaveCount(0);
});

test('shopping list on a phone: a compact − amount +, words not broken, the stepper below the text when very narrow', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 });
  await signIn(page, { sub: 'sina', name: 'Sina', admin: true });
  await page.goto('/todos');
  await page.getByLabel('Neue Liste').fill(unique('Schmal'));
  await page.getByRole('button', { name: 'Einkaufsliste anlegen' }).click();
  for (const t of ['Rinderbouillon', '1,5 kg Mehl']) {
    await page.getByLabel('Was fehlt?').fill(t);
    await page.getByLabel('Was fehlt?').press('Enter');
    await expect(page.getByRole('checkbox', { name: t })).toBeVisible();
  }
  await expect(page.locator('#todos-status')).toHaveText(''); // synced: no more redraws
  await page.evaluate(() => document.fonts.ready); // measured in IBM Plex, not the wider fallback
  const row = item(page, 'Rinderbouillon');
  const lines = () => row.locator('.todo-text').evaluate((el) => { const r = document.createRange(); r.selectNodeContents(el); return r.getClientRects().length; });
  const box = async (sel) => row.locator(sel).boundingBox();
  await expect.poll(async () => (await box('.shop-stepper'))?.width).toBeLessThanOrEqual(140);
  await expect.poll(lines).toBe(1); // "Rinderbouillon" on one line next to the stepper
  const amount = item(page, '1,5 kg Mehl').locator('.shop-amount');
  await expect(amount).toHaveText('1,5 kg');
  await expect.poll(() => amount.evaluate((el) => { const r = document.createRange(); r.selectNodeContents(el); return r.getBoundingClientRect().width <= el.clientWidth; })).toBe(true);

  await page.setViewportSize({ width: 340, height: 800 }); // a narrow phone: still beside, still one line
  await expect.poll(lines).toBe(1);

  await page.setViewportSize({ width: 320, height: 800 });
  await expect.poll(async () => {
    const t = await box('.todo-text');
    const st = await box('.shop-stepper');
    return st.y >= t.y + t.height - 1; // below the text, not beside it
  }).toBe(true);
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(320);
});
