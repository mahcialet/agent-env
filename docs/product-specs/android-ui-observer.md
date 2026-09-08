---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Android UI observer

[日本語](android-ui-observer.ja.md)

The observer operates only on an existing, confirmed lease-owned Android Emulator.
It sees Android accessibility semantics, including those exposed by Flutter, not
Flutter widgets. No target-project instrumentation or manifest changes are required.

## Selection and state

`ui` commands take a lease ID and either `--application NAME` or `--runtime NAME`.
An application selects its durable Android runtime and package. Without either
selector exactly one allocated Android runtime is required. Application snapshots
filter to that package by default; `--all-windows` explicitly includes system UI.
Runtime-only snapshots include all accessible windows. Missing, conflicting and
ambiguous selections fail before effects. Every operation holds the lease operation
fence and rechecks marker, process, console identity and explicit local ADB routing.

Snapshot, screenshot and logcat are allowed on ready/degraded, live owned runtimes.
Quarantined, expired, released or cleanup-requested leases are refused. Mutation
requires a ready, active, unexpired lease. Read-only requests may install the verified
observer companion on the disposable owned device; this effect is recorded.

## Commands

Use the existing `--output table|json` envelope. The proposed command surface is:

```text
agent-env ui snapshot LEASE --application mobile-app
agent-env ui screenshot LEASE --application mobile-app
agent-env ui tap LEASE --snapshot SNAPSHOT --node n7
agent-env ui set-text LEASE --snapshot SNAPSHOT --node n3 --text VALUE
agent-env ui tap-coordinate LEASE --runtime phone --x 120 --y 240
agent-env ui back LEASE --runtime phone
agent-env ui home LEASE --runtime phone
agent-env ui swipe LEASE --runtime phone --x 120 --y 400 --to-x 120 --to-y 100
agent-env ui wait LEASE --application mobile-app --contains Ready --timeout 10s
agent-env ui logcat LEASE --application mobile-app --since 30s
agent-env ui recover LEASE --run RUN
```

No arbitrary serial, shell command, selector expression or artifact path is accepted.
Timeouts are positive and capped at 60 seconds; swipe duration is 50–2000 ms. Wait
polls at most once per second and reports timeout without input or unbounded retry.

## Snapshot and action contract

Version 1 snapshots identify lease, runtime, serial, package scope, backend version,
capture time and snapshot ID. Windows and nodes retain class, package, resource ID,
label, text, bounds, relevant state/actions and parent context. Node references are
local to a snapshot. Bounds are device pixels. Stable traversal supplies compact
numbered text. Limits are 1000 nodes, depth 64, 4096 characters per field and 1 MiB
of response data; truncation is explicit. Android window IDs are observation-only
and do not form persistent action identity. Partial snapshots are diagnostic only.

A semantic action loads a registered snapshot with verified path and digest,
checks lease/runtime and recorded backend provenance, and reobserves under the same
operation fence. A snapshot from a different helper build cannot authorize input.
It matches the complete recorded semantic fingerprint against current nodes,
requires exactly one match, and dispatches through that current accessibility node.
The fingerprint includes semantic window identity (type/title/bounds/root package/class), ancestor context, class, package, resource ID,
noneditable label/text, bounds and action/state flags; it excludes ordinal refs and
editable/password values. Missing and duplicate matches produce stable
`AGENTENV-UI-STALE` / `AGENTENV-UI-AMBIGUOUS` diagnostics without input.
There is no coordinate fallback. Dynamic movement deliberately requires a fresh
snapshot. There is no atomicity guarantee against the app changing after validation.

Text replacement uses `ACTION_SET_TEXT` and requires a focused editable node that
advertises that action. Tap and take a fresh snapshot first if focus is required.
Success requires read-back equality, otherwise the result is uncertain and must not
be automatically retried. Accessibility refusal is not a successful mutation.
Coordinate tap is a separate explicit action. Back/Home and single-pointer swipe
use native Android input. Unsupported APIs report `AGENTENV-UI-UNAVAILABLE`.

## Evidence and privacy

Intent, target identity, completion or uncertainty, and artifact digests are durable.
Editable/password text and set-text payloads are fully redacted from retained JSON,
raw tree evidence, logs and errors. Read-back comparison happens inside the helper;
only the boolean result leaves it. Labels and noneditable text can still contain
secrets and receive configured secret redaction. Snapshots cannot guarantee that
arbitrary application text is nonsensitive. Evidence uses private file permissions.

Screenshots cover the entire display even when an application is selected. PNG
pixels cannot be text-redacted. A complete PNG is decoded, dimensions checked and
bounded to 16 MiB / 16 megapixels before atomic persistence and digest registration.

Logcat requires an application, scopes to its currently observed PID and reports
that PID and device-time lower bound. It does not claim historical coverage across
process restarts or subprocesses, never clears global logs, and never silently
falls back to unscoped capture. Output is redacted and capped at 256 KiB / 2000 lines, available inline in text/JSON
as well as a registered artifact. PID reuse can include historical lines from a
previous process; attribution is to the current numeric PID, not proven package
history.

## Companion and portability

A separate agent-env-owned, self-targeting instrumentation APK uses stable Android
platform `UiAutomation`; Android API 26 or later is required. It has no AndroidX or
target-app dependency. Its source, protocol version and build inputs belong to this
repository. An explicit native Go build tool produces an APK and metadata containing
its digest and source digest. Ordinary Go builds/tests require no Android SDK/JDK.
The runtime verifies this metadata and installed APK identity before using it;
a conflicting preinstalled helper is refused, not replaced. Removing the disposable
AVD removes the helper. No global ADB service or host tool configuration is changed.

An internal operation timeout may use up to ten additional seconds to stop only
this verified companion and confirm its absence, while the original lease fence
remains valid. Input is never retried. If the caller cancels or the fence is lost,
automatic recovery does not run. Interrupted/unconfirmed device operations retain
the existing running-command cleanup barrier. Destroy cannot race input; it may return busy while a bounded
observer operation owns the fence. Uncertain input must be investigated, never
silently retried. After an interruption, `ui recover LEASE --run RUN` can terminate a registered running
UI helper operation only when the registry already records positive
`termination-unconfirmed` recovery eligibility and the original result artifact is
registered and verifiable. Missing classification, including a crash before that
classification was saved, remains blocked. A failed classification write must never
make host-process or evidence uncertainty eligible for helper recovery. The command
uses a fresh fence, proves helper identity and process absence,
and retains recovery evidence before marking the original operation failed/uncertain.
It never recovers native input or arbitrary test processes, removes another run's
barrier, reconstructs lost screenshots, or reports the interrupted operation as
successful. Evidence failures still require retained completion/recovery evidence
before cleanup can proceed.
