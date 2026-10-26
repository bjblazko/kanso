// SPDX-License-Identifier: Apache-2.0
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: 'http://127.0.0.1:18091',
    locale: 'de-DE',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'desktop-chromium', use: { ...devices['Desktop Chrome'], locale: 'de-DE' } },
    { name: 'mobile-webkit', use: { ...devices['iPhone 15'], locale: 'de-DE' } },
  ],
  webServer: {
    command: './start-server.sh',
    url: 'http://127.0.0.1:18091/healthz',
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
