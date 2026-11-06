// SPDX-License-Identifier: Apache-2.0
// Shopping preview: changing servings or list refreshes the preview at once;
// without script the "Vorschau aktualisieren" button does the same.
for (const form of document.querySelectorAll('form[data-autosubmit]')) {
  form.addEventListener('change', () => form.requestSubmit());
}
