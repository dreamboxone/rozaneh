/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 * Adapted from Zirgozar status.js for the Rozaneh dashboard.
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


function icon(name) {
	var paths = {
		copy: '<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1"/>',
		trash: '<path d="M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7M14 10v7"/>',
		server: '<rect x="4" y="3" width="16" height="7" rx="2"/><rect x="4" y="14" width="16" height="7" rx="2"/><path d="M8 6h.01M8 17h.01M12 6h5M12 17h5"/>',
		dns: '<circle cx="12" cy="12" r="9"/><ellipse cx="12" cy="12" rx="4" ry="9"/><path d="M3 12h18M5 6h14M5 18h14"/>',
		settings: '<path d="M3 6h18M3 12h18M3 18h18"/><circle cx="8" cy="6" r="2"/><circle cx="16" cy="12" r="2"/><circle cx="10" cy="18" r="2"/>',
		power: '<path d="M12 3v9"/><path d="M6.4 6.4a8 8 0 1 0 11.2 0"/>',
		refresh: '<path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4M4 13a8 8 0 0 0 14.9 3M20 20v-4h-4"/>',
		download: '<g class="mk-download-arrow"><path d="M12 3v12M7 10l5 5 5-5"/></g><path d="M4 17v4h16v-4"/>',
		cpu: '<rect x="5" y="5" width="14" height="14" rx="2"/><path d="M9 9h6v6H9zM9 2v3m6-3v3M9 19v3m6-3v3M2 9h3m-3 6h3M19 9h3m-3 6h3"/>',
		shield: '<path d="m12 2 8 4v6c0 5-8 10-8 10S4 17 4 12V6z"/><path d="m8 12 3 3 5-6"/>',
		sun: '<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5m11 11L19 19M5 19l1.5-1.5m11-11L19 5"/>',
		moon: '<path d="M20 15.5A8.5 8.5 0 0 1 8.5 4 8.5 8.5 0 1 0 20 15.5z"/>',
		home: '<path d="m3 10 9-7 9 7v10a1 1 0 0 1-1 1h-5v-7H9v7H4a1 1 0 0 1-1-1z"/>',
		update: '<path d="M12 3v12m-5-5 5 5 5-5M4 16v5h16v-5"/>',
		log: '<path d="M14 2H5v20h14V7zM14 2v5h5M8 11h8M8 15h8M8 19h5"/>',
		status: '<path d="M3 12h4l3-8 4 16 3-8h4"/>',
		router: '<rect x="3" y="12" width="18" height="8" rx="2"/><path d="M7 12V5m10 7V5M6 3l1 2 1-2m8 0 1 2 1-2M7 16h.01M11 16h.01"/>'
		,send: '<path d="m22 2-7 20-4-9-9-4zM22 2 11 13"/>'
		,code: '<path d="m8 6-6 6 6 6m8-12 6 6-6 6m-3-15-2 18"/>'
	};
	var node = E('span', { 'class': 'mk-ico' + (name === 'download' ? ' mk-download' : ''), 'aria-hidden': 'true' });
	node.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' + (paths[name] || paths.status) + '</svg>';
	return node;
}

return baseclass.extend({brand: brand, icon: icon});
