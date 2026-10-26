// SPDX-License-Identifier: Apache-2.0
// Draws Mermaid diagrams. The library (vendored, without the ELK layout) is
// only fetched when a page actually contains a diagram.

const source = document.querySelector('script[data-mermaid-src]')?.dataset.mermaidSrc;
let mermaid;

function theme() {
  const forced = document.documentElement.dataset.theme;
  const dark = forced ? forced === 'dark' : window.matchMedia('(prefers-color-scheme: dark)').matches;
  return dark ? 'dark' : 'neutral';
}

async function load() {
  if (!mermaid) {
    mermaid = (await import(source)).default;
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: 'strict',
      layout: 'dagre',
      htmlLabels: false, // labels as SVG text: no HTML from page authors reaches the DOM
      flowchart: { htmlLabels: false },
      theme: theme(),
      fontFamily: '"IBM Plex Sans", system-ui, sans-serif',
    });
  }
  return mermaid;
}

function showError(node, error) {
  const message = document.createElement('p');
  message.className = 'diagram-error';
  message.textContent = `${node.dataset.errorLabel ?? 'Diagram error'}: ${error?.message ?? error}`;
  node.after(message);
}

async function render(root) {
  const nodes = [...root.querySelectorAll('pre.mermaid:not([data-processed])')];
  if (!nodes.length || !source) return;
  const m = await load();
  for (const node of nodes) {
    try {
      await m.run({ nodes: [node] });
    } catch (error) {
      node.dataset.processed = 'error';
      showError(node, error);
    }
  }
}

document.addEventListener('kanso:rendered', (event) => render(event.detail));
render(document);
