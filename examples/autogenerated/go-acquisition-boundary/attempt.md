# Go acquisition boundary experiment

Purpose: settle transport ownership and Android integration for
SUN-WIFI-GO-ACQUISITION-01 with executable Go evidence, before a production
engine or APK is claimed.

The user explicitly chose Go. No Idriç fallback attempt is appropriate for
the engine. The existing typed Idriç DEX writer remains the candidate owner
for mechanical Android callback classes.

## Values, operations and effects

- A `Network` is an Android network handle plus a local connection generation
  and an observed Wi-Fi transport flag. A handle alone is not authority or a
  current connectivity result.
- A `BoundDialer` is supplied by the Android adapter. It must resolve via the
  selected Android Network and bind every outgoing socket to it. The host test
  supplies a named loopback fixture; it is not Android binding evidence.
- An `Engine` owns one connection generation and its HTTP pool. Replacing a
  generation cancels its requests and makes late results ineligible.
- `Probe(ctx)` returns first-response status, possible redirect, bounded body
  evidence, elapsed time and connection-reuse observations. It never follows
  portal redirects or disables TLS verification. Success means the exact
  configured endpoint contract passed on this generation, not unrestricted
  internet or completed portal authorization.
- Byte accounting wraps the socket, so TLS records and HTTP bytes are observed
  at the socket boundary. It excludes TCP/IP/Wi-Fi headers, ACKs and retries.
- `Capture(ctx, executable, args, limit)` executes a fixed read-only diagnostic
  with bounded output and elapsed time. Caller termination and remote Shizuku
  termination remain separate evidence boundaries.

Required invariants: missing/non-Wi-Fi identity never starts a request; every
network replacement gets a fresh pool; stale completion never becomes current;
TLS and endpoint contracts remain checked; body/header/time budgets bound
portal failures; no automatic join or login; no network credentials.

Go modules and tests here are bounded experiments, not a second scanner or a
production API promise. Production history/ranking/UI are Earth follow-up work.

## Planned checks and exact limits

Host loopback TLS tests will exercise pool reuse, redirect refusal, body limits,
deadline cancellation after headers, generation cancellation, missing binding,
wrong transport, wrong TLS signer and endpoint mismatch. Diagnostic tests will
exercise stdout/stderr floods and hangs. Android ARM64 cross-compilation will
be attempted using the pinned Go compiler. Runtime, Binder/framework callback,
network binding and physical C67 execution remain NOT_RUN until exercised.

Toolchain: official go1.27.3 linux-amd64 archive, SHA256
`65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`.
Results and the chosen architecture will be recorded in docs/go-acquisition/.

## Result

Host protocol/cancellation tests, original scanner fixtures, vet and the
loopback benchmark ran successfully. The Android-specific read-only executable
cross-compiles without cgo; it does not use Go DNS or prove Android networking.

Exact-code hosted run 41753953052 at
`ff83e51d1e02e6b5f3a28959cd0f6f81f7961ecc` passes the host C-to-Go ABI observer,
race detector and Android API34 ARM64 Go shared-library link/export checks.
The local tested source tree and remote code tree are identical
(`b96adeb289573a72792f60126fdd784d76d899f0`). The CLI push transport lacked
credentials; the connected repository write published the same verified tree.

No compiler gap was found in Android-NDK's generic native framework-class
writer. Its separate Activity directive parser is narrower and must not be
misidentified as the whole compiler. No compiler/language change is justified.
The missing production work is framework/Go integration plus cancellable
per-Network DNS/socket implementation and its runtime qualification, not a
reason to replace the chosen language. See docs/go-acquisition/decision.md and
implementation-jobs.md for exact obligations. Physical C67 remains NOT_RUN.
