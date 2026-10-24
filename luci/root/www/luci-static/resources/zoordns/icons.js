/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 * Adapted from Zirgozar status.js for the zoorDNS dashboard.
 * https://github.com/dreamboxone/zirgozar
 */
'use strict';
'require baseclass';
var BRANDS = {
	instagram:  '<path fill-rule="evenodd" d="M7 2h10a5 5 0 0 1 5 5v10a5 5 0 0 1-5 5H7a5 5 0 0 1-5-5V7a5 5 0 0 1 5-5zm0 2a3 3 0 0 0-3 3v10a3 3 0 0 0 3 3h10a3 3 0 0 0 3-3V7a3 3 0 0 0-3-3H7zm5 3.5a4.5 4.5 0 1 1 0 9 4.5 4.5 0 0 1 0-9zm0 2a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5z"/>' +
	            '<circle cx="17.5" cy="6.5" r="1.2"/>',
	reddit:     '<path fill-rule="evenodd" d="M4 14.5a8 5.5 0 1 0 16 0a8 5.5 0 1 0-16 0zm3.7-1a1.4 1.4 0 1 0 2.8 0a1.4 1.4 0 1 0-2.8 0zm5.8 0a1.4 1.4 0 1 0 2.8 0a1.4 1.4 0 1 0-2.8 0zM8.8 16.6q3.2 2.2 6.4 0q-3.2 1.1-6.4 0z"/>' +
	            '<circle cx="4.6" cy="10.6" r="1.9"/><circle cx="19.4" cy="10.6" r="1.9"/><circle cx="18.6" cy="4.4" r="1.7"/>' +
	            '<path d="M12 9l1.3-5.2 4 .9" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>',
	x:          '<path d="M18.901 1.153h3.68l-8.04 9.19L24 22.846h-7.406l-5.8-7.584-6.638 7.584H.474l8.6-9.83L0 1.154h7.594l5.243 6.932ZM17.61 20.644h2.039L6.486 3.24H4.298Z"/>'
};

function brand(name) {
	var s = E('span', { 'class': 'mk-ico', 'aria-hidden': 'true' });
	s.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor" stroke="none">' + (BRANDS[name] || '') + '</svg>';
	return s;
}


return baseclass.extend({brand: brand});
