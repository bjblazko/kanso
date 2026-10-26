// SPDX-License-Identifier: Apache-2.0
import { test, expect } from '@playwright/test';
import { signIn, unique, createSpace, createPage } from './helpers.js';

const fence = (lines) => ['```mermaid', ...lines, '```', ''].join('\n');

async function pageWith(page, sub, markdown) {
  await signIn(page, { sub, name: sub, admin: true });
  await createSpace(page, unique(`Diagramme ${sub}`));
  await createPage(page, 'Abläufe');
  await page.getByRole('link', { name: 'Bearbeiten' }).click();
  await page.getByRole('textbox', { name: 'Quelltext' }).fill(markdown);
  await page.getByRole('button', { name: 'Speichern' }).click();
  await expect(page).toHaveURL(/\/p\/[a-z2-7]{12}$/);
}

function collectCSP(page) {
  const violations = [];
  page.on('console', (msg) => { if (/Content.Security.Policy/i.test(msg.text())) violations.push(msg.text()); });
  return violations;
}

test('mermaid: flowchart and sequence diagram render without CSP violations', async ({ page }) => {
  const csp = collectCSP(page);
  await pageWith(page, 'mara', fence(['flowchart TD', '  A[Start] --> B{Prüfen}', '  B -->|ok| C[Fertig]'])
    + '\n' + fence(['sequenceDiagram', '  Anna->>Ben: Einkaufen?', '  Ben-->>Anna: Ja']));
  const diagrams = page.locator('.prose pre.mermaid');
  await expect(diagrams.nth(0).locator('svg')).toBeVisible();
  await expect(diagrams.nth(1).locator('svg')).toBeVisible();
  expect(csp).toEqual([]);
});

test('mermaid: HTML in labels never runs', async ({ page }) => {
  await pageWith(page, 'malte', fence(['flowchart TD', '  A --> B["<img src=x onerror=window.__xss=1>"]']));
  await expect(page.locator('.prose pre.mermaid svg')).toBeVisible();
  await expect(page.locator('.prose img[src="x"]')).toHaveCount(0);
  expect(await page.evaluate(() => window.__xss)).toBeUndefined();
});

test('mermaid: the ELK layout is not shipped and fails gracefully', async ({ page }) => {
  await pageWith(page, 'mila', fence(['flowchart-elk TD', '  X --> Y']));
  await expect(page.getByRole('heading', { level: 1, name: 'Abläufe' })).toBeVisible();
  await expect(page.locator('.prose pre.mermaid svg').or(page.locator('.diagram-error'))).toBeVisible();
});
