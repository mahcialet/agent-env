---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Browser/CDP design

[日本語](browser-cdp-automation.ja.md)

The [product contract](../product-specs/browser-cdp-automation.md) defines commands
and limits. The [active ExecPlan](../exec-plans/active/browser-cdp-automation.md)
records implementation decisions and direct native evidence.

## Ownership and dependencies

Config owns explicit browser bindings above generic process configuration. Domain
owns serializable binding, identity, page, snapshot, node and observation values.
`app.BrowserProvider.Observe` accepts a runtime, binding, typed request and native
ownership callback. App owns selection, operation fencing, persisted run intent,
snapshot provenance, redaction and registered artifacts. `internal/browser/cdp`
implements browser-specific discovery, protocol validation and typed operations;
its CDP transport attaches with `github.com/gorilla/websocket` v1.5.3. It does not
import a concrete runtime adapter or start/terminate browsers. CLI wires the
provider and parses/renders commands. Store uses existing run/artifact records;
no browser lifecycle table or new process owner is introduced.

## Connection authority

App first inspects the immutable persistent-process identity under the lease
fence. The adapter verifies the expanded, recorded singleton debugging/profile
switches, resolves the exact reserved loopback port and performs bounded discovery
without HTTP redirects or an ambient proxy. A browser-level WebSocket must match
that loopback authority. Discovery and `Browser.getVersion` must agree on product
and protocol. `SystemInfo.getProcessInfo` must identify the browser PID as the
recorded native root, and `Browser.getBrowserCommandLine` must confirm the private
profile, automation and debugging settings. Rechecking native ownership closes
ordinary exit/reuse transitions before protocol effects. This is accidental
isolation for trusted repositories, not protection from a malicious same-user CDP
server forging protocol responses.

## Pages, sessions and stale references

Requests use a browser-level connection and target sessions, bounded deadlines,
cancellation and event demultiplexing. Page selection is deterministic and refuses
ambiguous implicit selection. Snapshot identity includes the lease, browser,
native PID/birth/port/WebSocket, page, document loader and backend DOM/frame
fingerprint. A digest-verified registered snapshot is required for semantic input;
filesystem lookalikes, cross-lease references, truncation and stale documents do
not authorize it. Fresh frame/node state must match immediately before input;
there is no fallback to stored coordinates. Same-origin iframe and shadow DOM
remain observable. Cross-origin/OOPIF observation and all iframe semantic input
are explicitly unsupported in this slice.
Fixed internal node readback is permitted, but unrestricted public evaluation is
not. No shared Browser/Android UI abstraction is introduced prematurely.

## Effects, diagnostics and privacy

Every operation holds the lease fence through protocol work and evidence
finalization. Mutations write run intent first, attempt input once and retain an
unfinished-run barrier when completion or evidence cannot be confirmed. A lost
fence cannot finalize successor-owned state. Native process termination and
profile removal remain the existing process adapter's responsibility.

AX is the primary semantic representation. DOM/layout evidence is a bounded
supplement; editable/password values are suppressed. PNG validation checks image
integrity, size and dimensions before publication. Secret-bearing inputs are
transient and structured string redaction preserves valid JSON. Network evidence
stores no headers or bodies; console and network attachment windows are bounded.
Screenshots and unknown page text may contain secrets and remain private evidence.
This slice does not offer persistent background diagnostics, downloads, browser
restart, external attachment or public CDP passthrough.

## Validation strategy

Negative config fixtures establish explicit binding and exact switch requirements.
App tests must cover ownership rechecks, stale/cross-lease references, run/evidence
failure and destroy fencing. Transport fixtures must reject foreign discovery and
malformed/oversized responses. Real native tests use pinned Chrome for Testing
152.0.7977.82 on Windows, macOS and Linux with Go 1.27; this is the selected matrix,
not a claim that native acceptance has passed. Browser/backend fixtures exercise
an endpoint owned by the same lease. Record actual product/protocol versions and
native results in the plan; cross-compilation alone does not validate CDP behavior.

Explicit navigation invalidates earlier browser snapshots even for hash/history
changes. Document tokens also include a raw-URL digest (not its query text), and
node fingerprints include allowlisted nontext AX state flags. Set-text requires
an explicit `--text`; an explicit empty string clears the field. CLI validates
required operation arguments and refuses explicit zero durations before opening
the store.
