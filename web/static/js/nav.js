// SPDX-License-Identifier: Apache-2.0
// On narrow screens the navigation starts folded, so the page comes first.
const nav = document.querySelector('.sidenav > details');
if (nav && window.matchMedia('(max-width: 59.99rem)').matches) nav.open = false;
