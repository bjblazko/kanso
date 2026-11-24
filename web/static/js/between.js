// SPDX-License-Identifier: Apache-2.0
// Zwischendurch: another lifting step on request, without losing what was
// typed; and the reminder starts counting again (nudge.js), since this is the break.
const tips = [...document.getElementById('between-tips').content.children];
const button = document.getElementById('between-another');
let shown = 0;

button.hidden = tips.length < 2;
button.addEventListener('click', () => {
  shown = (shown + 1) % tips.length;
  document.getElementById('between-tip-title').textContent = tips[shown].dataset.title;
  document.getElementById('between-tip-body').textContent = tips[shown].textContent;
});
window.KansoNudge?.reset();
