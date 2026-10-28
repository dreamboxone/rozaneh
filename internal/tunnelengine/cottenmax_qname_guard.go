// SPDX-License-Identifier: Apache-2.0
// Adapted from WhiteDNS/CottenDNS revision 7db7b73658c791cce62300d29647e4d369d6e1e4.
// CottenDNS modifications Copyright (c) 2026 Amin Mahmoudi, NullRoute1970, tajirax.
// Rozaneh modifications Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
// See licenses/CottenDNS-LICENSE and internal/tunnelengine/NOTICE.cottendns.txt.
package soar

import "time"

// Optional experimental guard. Called only after this original question's downstream
// packet has passed session authentication. An ordinary short poll, forged
// response or DNS error cannot keep a large upload-name estimate alive.
func (c *Client) noteAuthenticatedLargeQName(q *pendingQuery, now time.Time) {
	end, err := skipName(q.question, 0)
	if err != nil || end+4 != len(q.question) {
		return
	}
	// Locally generated questions use uncompressed names. Their presentation
	// length is wire length minus the first label-length byte and root byte.
	nameLen := end - 2
	// A medium name does not prove the current cap: a 150-character cutoff
	// must not be hidden by valid 140-character replies while the cap is 253.
	if nameLen > c.cfg.SafeQName && nameLen >= q.p.nameLen-c.qnameProofSlack() {
		q.p.lastLargeNameAuth = now
		q.p.lastLargeNameLen = nameLen
	}
}

func (c *Client) qnameProofSlack() int {
	if c.z.labelJitter {
		return 5 // base32 rounding plus three reserved separators
	}
	return 2 // base32 rounding
}

func (p *path) freshLargeQNameProof(now time.Time, capSlack int) bool {
	if p.lastLargeNameAuth.IsZero() || p.lastLargeNameLen < p.nameLen-capSlack {
		return false
	}
	age := now.Sub(p.lastLargeNameAuth)
	return age >= 0 && age < p.rto()
}
