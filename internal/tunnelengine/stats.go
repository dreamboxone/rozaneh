// SPDX-License-Identifier: Apache-2.0
// Adapted from WhiteDNS/CottenDNS revision 7db7b73658c791cce62300d29647e4d369d6e1e4.
// CottenDNS modifications Copyright (c) 2026 Amin Mahmoudi, NullRoute1970, tajirax.
// Rozaneh modifications Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
// See licenses/CottenDNS-LICENSE and internal/tunnelengine/NOTICE.cottendns.txt.
package soar

import "time"

// PathStat is a snapshot of one resolver path, for diagnostics: which
// resolvers carry the tunnel on the user's network, and how well.
type PathStat struct {
	Resolver        string        `json:"resolver"`            // host:port of the recursive resolver
	Transport       string        `json:"transport"`           // "udp" or "tcp" (TCP/53, or what Config.DialTCP makes of it)
	Verified        bool          `json:"verified"`            // has carried at least one authenticated answer
	Healthy         bool          `json:"healthy"`             // currently usable with recent authenticated progress
	Suspended       bool          `json:"suspended"`           // backed off after a run of losses
	Candidate       bool          `json:"candidate,omitempty"` // added by discovery
	RTT             time.Duration `json:"rtt"`
	Loss            float64       `json:"loss"` // fraction lost over the recent outcome window
	Inflight        int           `json:"inflight"`
	Window          float64       `json:"window"`
	NameLen         int           `json:"name_len"`          // current query-name cap
	ObservedNameLen int           `json:"observed_name_len"` // largest authenticated query name
	ObservedAnswer  int           `json:"observed_answer"`   // largest authenticated DNS reply, wire bytes
	AnswerLimit     int           `json:"answer_limit"`      // answer size cap sizing has accepted for this path; 0 until verified
	RecordType      string        `json:"record_type"`
	Successes       int           `json:"successes"`
	LastSuccess     time.Time     `json:"last_success,omitzero"`
}

// PathStats returns a snapshot of every path, in the order paths were added: configured
// resolvers first, then paths added later (TCP siblings under TCP "auto", and discovered
// resolvers).
func (c *Client) PathStats() []PathStat {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	out := make([]PathStat, 0, len(c.paths))
	for _, p := range c.paths {
		s := PathStat{
			Resolver:        p.addr.String(),
			Transport:       "udp",
			Verified:        p.verified,
			Healthy:         p.healthy(now),
			Suspended:       p.suspended.After(now),
			Candidate:       p.candidate,
			Loss:            p.lossRate(),
			Inflight:        p.inflight,
			Window:          p.cwnd,
			NameLen:         p.nameLen,
			ObservedNameLen: p.observedNameLen,
			ObservedAnswer:  p.observedAnswer,
			RecordType:      "TXT",
			Successes:       p.successes,
			LastSuccess:     p.lastSuccess,
		}
		if p.tcp != nil {
			s.Transport = "tcp"
		}
		if p.hasRTT {
			s.RTT = p.srtt
		}
		if p.verified {
			s.AnswerLimit = 512 + 48*p.sizeOK
		}
		if p.qtype == typeAAAA {
			s.RecordType = "AAAA"
		}
		out = append(out, s)
	}
	return out
}

// Established reports whether a session with the server is up.
func (c *Client) Established() bool {
	c.mu.Lock()
	cs := c.sess
	c.mu.Unlock()
	if cs == nil {
		return false
	}
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case <-cs.ready:
		return cs.session != nil && !cs.isClosed()
	default:
		return false
	}
}
