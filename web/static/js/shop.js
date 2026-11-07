// SPDX-License-Identifier: Apache-2.0
// Shopping preview: changing servings or list refreshes the preview at once
// (without script the "Vorschau aktualisieren" button does it); the button
// that adds to the list works once, so a double tap adds nothing twice.
const form = document.querySelector('form[data-shop]');
const refresh = form?.querySelector('[data-refresh]');
const take = form?.querySelector('[data-take]');

form?.addEventListener('change', (event) => {
  if (event.target.matches('#servings, #list')) form.requestSubmit(refresh);
});
form?.addEventListener('submit', (event) => {
  if (event.submitter === take) take.disabled = true;
});
