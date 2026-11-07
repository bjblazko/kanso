// SPDX-License-Identifier: Apache-2.0
// A button with data-wait says, once pressed, that the answer can take a while,
// and cannot be pressed twice.
for (const form of document.querySelectorAll('form')) {
  form.addEventListener('submit', (event) => {
    const button = event.submitter;
    if (!button?.dataset.wait) return;
    button.textContent = button.dataset.wait;
    setTimeout(() => { button.disabled = true; }); // after the form data is taken
  });
}
