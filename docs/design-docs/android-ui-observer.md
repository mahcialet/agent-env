---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Android UI observer design

[日本語](android-ui-observer.ja.md)

The [product contract](../product-specs/android-ui-observer.md) defines behavior.
The [active plan](../exec-plans/active/android-ui-observer.md) records implementation
and acceptance evidence. This adds no target-manifest section or SQL migration.

App owns selection, policy, operation fencing, durable intent and final artifacts
through a narrow `AndroidUIProvider` interface. Domain owns serializable observer
values. The existing Android adapter owns device effects and reuses `applicationADB`
identity/routing checks; it does not import Flutter or another runtime adapter.
CLI only parses arguments, wires providers and renders results.

Use existing `CommandRun` rows as intent and cleanup barriers. Acquire a context-bound
lease lock before loading state. Propagate that context to every registry and
provider operation. Write required evidence atomically and register digests before
publishing a terminal run. Cancellation, lock loss, uncertain process completion and
incomplete evidence do not clear the durable barrier. Text payloads never appear in
persisted argv. Final evidence may use the existing fenced WithoutCancel pattern.

Registered snapshot artifacts are read only through their expected lease artifact
root, with regular-file, containment, size and digest validation. Cross-lease refs
cannot become device input. Matching happens inside one companion invocation:
reobserve, compare semantic fingerprint, reject zero/multiple candidates, then act
through the selected live AccessibilityNodeInfo. Truncated observations cannot
prove uniqueness and cannot authorize semantic mutations. Editable values are
excluded from fingerprints and all serialized output. Preserve target scope and
runtime identity from the referenced snapshot even when no CLI selector is given.

The companion is self-targeting instrumentation, independent of the target app.
Platform UiAutomation exposes Flutter semantics without an AndroidX dependency.
The backend spike proved that merely accepting ACTION_SET_TEXT is insufficient:
an unfocused Flutter node can return true without updating text. Require advertised
capability/focus and verify replacement via fresh read-back before reporting success.
Keyboard appearance also changes traversal ordinals; never use those as identity.

Build the companion explicitly with native Go orchestration of javac, aapt2, Java D8,
ZIP and Java apksigner. Avoid SDK shell/batch wrappers in the core workflow. Include
version/source/APK digests in provenance, verify installed digest, and reject foreign
helper packages. Helper distribution is opt-in local build; no automatic downloads,
license acceptance or mandatory JVM for ordinary Go consumers. API 26 is the
initial floor; actual SDK evidence must identify the tested API rather than imply
coverage of every supported Android image.

Unit/fake-adapter tests cover selection, stale/ambiguous refs, privacy, evidence
failures and fenced cleanup. Native OS CI validates argv, filesystem and persistence
behavior. Real Linux Emulator tests independently prove Flutter semantics, Unicode
replacement, PNG evidence and sibling isolation. Cross-builds are not SDK evidence.

Recovery preserves the lease fence. Internal helper deadlines may trigger bounded
quiescence under the original still-active operation context, following a fresh
fenced intent write. Caller cancellation or lock loss retains the running barrier.
Explicit `ui recover --run` handles a registered interrupted helper run under a new
fence. Eligibility must be positively recorded as `termination-unconfirmed`, with
a registered, digest-verified original result. The absence of a forbidden
classification is insufficient: a failure while saving the final classification
can leave the earlier run intent in the registry. Unclassified crash interruptions
therefore remain blocked. Recovery validates stored runtime/serial, verifies the installed companion digest,
stops only that package, confirms PID absence and records the failed/uncertain
outcome plus recovery artifacts before releasing its cleanup barrier. It never
retries input or recovers arbitrary native commands.

Android window IDs are ephemeral across instrumentation connections. Retain them
in raw observations, but exclude them from node matching; use semantic window
metadata instead. An unchanged Flutter tree must remain actionable after reconnect.
Duplicate semantic window identities still require unique node matching.
