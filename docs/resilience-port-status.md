# Resilience engine integration status

The development tree includes the CottenDNS engine pinned to
`7db7b73658c791cce62300d29647e4d369d6e1e4`. Its source notice identifies
the original Soar base as v0.2.3. This is a derivative engine, not an official
Soar v0.2.4 upgrade. Licenses and source notices are retained in
`internal/tunnelengine`, `licenses`, and the root NOTICE.

Implemented development controls:

- `-fec-download`, `-fec-data`, `-fec-parity`: authenticated download FEC
  negotiation. Both endpoints must support it. Disabled by default.
- `-qname-loss-guard`: authenticated query-size proof guard. Experimental,
  disabled by default.
- `-standby-domains`, `-failover-after`: lazy authenticated fallback for new
  connections, with the same server key. Existing streams are never replayed.
- Matching server command: `go build ./cmd/rozaneh-server`.

The existing source-import updater must not be treated as updating this local
derivative: updating the go.mod dependency does not change these engine files.
Release manifest generation and verification must be adapted before publication.

Validation completed: internal FEC tests, scheduler tests, authenticated standby
transfer, stream capacity and confirmed-connection tests, and Linux server builds.

Release blockers: the impaired-resolver integration test and the concurrent-loss
reproduction fail. These failures remain visible in the test suite. Do not publish
or deploy this development engine as a stability fix. Verify interoperability with
the deployed server before enabling FEC. No router or server deployment has been
performed for this engine port.
