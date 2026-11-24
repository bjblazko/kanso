// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn } from './helpers.js';

/** Replaces Notification and AudioContext with recorders, so a test can see what rang. */
async function stubAlerts(page, permission = 'granted') {
  await page.addInitScript((perm) => {
    window.__notes = [];
    window.__beeps = 0;
    class FakeNotification {
      constructor(title, options) { window.__notes.push(options.body); }
      static permission = perm;
      static requestPermission() { FakeNotification.permission = perm; return Promise.resolve(perm); }
    }
    if (perm !== 'default') FakeNotification.permission = 'default';
    window.Notification = FakeNotification;
    class FakeAudio {
      constructor() { this.currentTime = 0; this.destination = {}; this.state = 'running'; }
      resume() { return Promise.resolve(); }
      createOscillator() { return { type: '', frequency: { value: 0 }, connect() {}, start() { window.__beeps += 1; }, stop() {} }; }
      createGain() {
        return { gain: { value: 0, setValueAtTime() {}, linearRampToValueAtTime() {}, exponentialRampToValueAtTime() {} }, connect() {} };
      }
    }
    window.AudioContext = FakeAudio;
  }, permission);
}

async function open(page, permission) {
  await page.clock.install({ time: new Date('2026-11-24T08:00:00') });
  await page.clock.pauseAt(new Date('2026-11-24T08:00:01')); // time moves only when the test says so
  await stubAlerts(page, permission);
  await signIn(page, { sub: 'fritz', name: 'Fritz' });
  await page.goto('/focus');
  await page.evaluate(() => localStorage.clear());
  await page.reload();
}

const time = (page) => page.locator('#focus-time');
const phase = (page) => page.locator('#focus-phase');

test('a focus round rings, notifies and waits at the break', async ({ page }) => {
  await open(page, 'granted');
  await expect(time(page)).toHaveText('25:00');
  await page.getByRole('button', { name: 'Start' }).click();
  await page.clock.fastForward('10:00');
  await expect(time(page)).toHaveText('15:00');
  await expect(page).toHaveTitle(/^15:00 · Fokus/);
  await page.clock.fastForward('15:01');
  await expect(phase(page)).toHaveText('Kurze Pause');
  await expect(time(page)).toHaveText('05:00');
  expect(await page.evaluate(() => window.__notes)).toEqual(['Fokus vorbei – Zeit für eine Pause.']);
  expect(await page.evaluate(() => window.__beeps)).toBeGreaterThan(0);
  await expect(page.getByRole('button', { name: 'Start' })).toBeVisible();
  await expect(page.locator('#focus-notify')).toHaveText('Am Ende: Ton und Benachrichtigung.');
  await expect(page.locator('#focus-break')).toBeVisible();
  await expect(page.locator('#focus-break-title')).not.toBeEmpty();
  await page.getByRole('button', { name: 'Überspringen' }).click();
  await expect(page.locator('#focus-break')).toBeHidden();
});

test('parked thoughts stay on the device and become a note', async ({ page }) => {
  await open(page, 'granted');
  await expect(page.getByRole('button', { name: 'Als Notiz' })).toBeDisabled();
  await page.getByLabel('Gedanken-Parkplatz').fill('Zahnarzt anrufen');
  await page.reload();
  await expect(page.getByLabel('Gedanken-Parkplatz')).toHaveValue('Zahnarzt anrufen');
  await page.getByRole('button', { name: 'Als Notiz' }).click();
  await expect(page).toHaveURL(/\/focus\?saved=/);
  await expect(page.getByLabel('Gedanken-Parkplatz')).toHaveValue('');
  await page.getByRole('link', { name: 'Gespeichert – zur Notiz' }).click();
  await expect(page.locator('main')).toContainText('Zahnarzt anrufen');
});

test('a reload keeps the running time', async ({ page }) => {
  await open(page, 'granted');
  await page.getByRole('button', { name: 'Start' }).click();
  await page.clock.fastForward('10:00');
  await page.reload();
  await expect(time(page)).toHaveText('15:00');
  await expect(page.getByRole('button', { name: 'Pause' })).toBeVisible();
  await page.getByRole('button', { name: 'Pause' }).click();
  await page.clock.fastForward('05:00');
  await expect(time(page)).toHaveText('15:00');
  await expect(page.getByRole('button', { name: 'Weiter' })).toBeVisible();
});

test('after four focus rounds comes the long break; reset starts over', async ({ page }) => {
  await open(page, 'granted');
  for (let i = 0; i < 7; i += 1) await page.getByRole('button', { name: 'Überspringen' }).click();
  await expect(phase(page)).toHaveText('Lange Pause');
  await expect(time(page)).toHaveText('15:00');
  await expect(page.locator('#focus-round')).toHaveText('Runde 4 von 4');
  expect(await page.evaluate(() => window.__notes)).toEqual([]);
  await page.getByRole('button', { name: 'Zurücksetzen' }).click();
  await expect(phase(page)).toHaveText('Fokus');
  await expect(page.locator('#focus-round')).toHaveText('Runde 1 von 4');
});

test('notifications denied: only the sound, no error', async ({ page }) => {
  const errors = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await open(page, 'denied');
  await page.getByLabel('Fokus', { exact: true }).fill('1');
  await page.getByLabel('Fokus', { exact: true }).blur();
  await expect(time(page)).toHaveText('01:00');
  await page.getByRole('button', { name: 'Start' }).click();
  await page.clock.fastForward('01:01');
  await expect(phase(page)).toHaveText('Kurze Pause');
  expect(await page.evaluate(() => window.__notes)).toEqual([]);
  expect(await page.evaluate(() => window.__beeps)).toBeGreaterThan(0);
  await expect(page.locator('#focus-notify')).toContainText('nur ein Ton');
  expect(errors).toEqual([]);
});
