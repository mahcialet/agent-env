---
status: completed
owner: maintainers
last_verified: 2026-09-09
---

# Add lease-owned Browser/CDP automation and semantic snapshots

[日本語](browser-cdp-automation.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `feat/browser-cdp-automation`.

PR #9 (`feat: add lease-owned persistent native process runtime`) is a hard
prerequisite. Begin only after PR #9 is merged into `master`.

Starting revision: `3837c12c88d880c3082eb9715795ae035de8a181`

## Purpose / Big Picture

After this work, an agent can run an isolated Chromium-family browser in a lease
and operate it through Chrome DevTools Protocol (CDP), without attaching to the
user's normal browser profile.

PR #9 remains the browser process owner:

    persistent process runtime
      -> native process identity
      -> private runtime directory
      -> dynamic loopback port
      -> logs/readiness/reconcile
      -> conservative destroy

Browser/CDP adds only browser-specific observation/control:

    Browser/CDP
      -> browser and target identity
      -> page/tab selection
      -> Accessibility semantic snapshot
      -> DOM/layout snapshot
      -> screenshot
      -> navigation
      -> stale-safe click/text/key/scroll
      -> waits
      -> console diagnostics
      -> bounded network capture

A target manifest may converge on:

```yaml
runtimes:
  browser-process:
    type: process
    source: app
    working_directory: .
    command:
      - chrome
      - --headless=new
      - --enable-automation
      - --user-data-dir=${runtime_dir}/profile
      - --remote-debugging-address=127.0.0.1
      - --remote-debugging-port=${port:cdp}
      - about:blank
    ports:
      cdp: {protocol: tcp}

browsers:
  web:
    type: chromium-cdp
    runtime: browser-process
    cdp_port: cdp
```

This syntax is finalized by the decisions below after inspecting merged PR #9. Do not infer a browser merely from a process exposing a port named
`cdp`.

Target workflow:

    agent-env create . --stack browser
    agent-env browser pages <lease> --browser web
    agent-env browser navigate <lease> --browser web --url http://127.0.0.1:...
    agent-env browser snapshot <lease> --browser web
    agent-env browser screenshot <lease> --browser web
    agent-env browser click <lease> --snapshot <snapshot-id> --node n4
    agent-env browser set-text <lease> --snapshot <snapshot-id> --node n2 --text ...
    agent-env browser console <lease> --browser web
    agent-env destroy <lease>

## Scope

In scope:

- Chromium-family browsers controlled with CDP;
- explicit browser binding to one owned `process` runtime and named TCP port;
- private browser profile under `${runtime_dir}`;
- loopback-only CDP;
- browser/product/protocol discovery;
- browser-level WebSocket and target sessions;
- page listing/select/create/close where safe;
- URL navigation;
- Accessibility tree semantic snapshots;
- DOMSnapshot-based DOM/layout evidence;
- compact text plus structured JSON snapshots;
- PNG screenshots;
- ephemeral snapshot-scoped node handles;
- stale-node revalidation before semantic actions;
- click, Unicode text replacement, key press and scroll;
- bounded waits;
- bounded console diagnostics;
- bounded network capture;
- iframe/shadow-DOM fixtures;
- operation fencing against destroy;
- cross-lease/browser/page isolation;
- profile privacy and evidence rules;
- real native Windows/macOS/Linux headless browser tests;
- Chrome for Testing as preferred CI evidence;
- bilingual durable documentation.

Out of scope:

- Firefox/WebDriver BiDi;
- Safari/WebKit;
- Playwright/Selenium/ChromeDriver as runtime requirements;
- attaching to arbitrary existing browsers;
- attaching to the user's default profile;
- Chrome DevTools MCP as a dependency;
- auto-downloading or bundling Chrome;
- extensions and reusable login profiles;
- arbitrary raw CDP passthrough;
- unrestricted public JavaScript evaluation;
- video/screencast;
- OCR/computer-vision selectors;
- visual-regression baseline management;
- persistent unbounded network tracing;
- browser auto-restart;
- remote browser hosts;
- browser security sandbox guarantees.

## Progress

### PR #10 review follow-up (2026-09-09)

- [x] Validate bounded Windows sharing-violation cleanup after native process-tree absence, preserving generic process ownership and failure barriers.

This plan was reopened on `feat/browser-cdp-automation` after Windows native run
34236523326 failed following the initial archive. The unchanged PR-triggered pass
34240370827 was not treated as a repair. The plan remained active through the
review fixes and a further Windows cleanup failure. All final gates passed at
`391288c`, and all nine review threads were replied to and resolved before this
bilingual re-archive on 2026-09-09.

- [x] Classify iframe access by browser-reported security origins; cover inherited, blob and opaque origins and the native timing regression.
- [x] Require a nonempty URL wait substring and reject role on URL waits.
- [x] Bound every persisted network string and mark DOM-name truncation.
- [x] Reject disappearance conclusions from truncated snapshots.
- [x] Subscribe only to required capture events; ignore unrelated events without disconnecting ordinary operations.
- [x] Persist semantic page/snapshot/node provenance before input, including uncertain runs, without text disclosure.
- [x] Correct bilingual architecture status and update contract/decision evidence.
- [x] Complete regression/race/harness and real three-OS native CI; reconcile final evidence before archival.
- [x] Reply to and resolve every addressed PR #10 review thread.

- [x] Merge PR #9 and record exact `master` revision.
- [x] Create `feat/browser-cdp-automation`.
- [x] Establish baseline validation evidence before implementation: full
  `go test -race ./...` passed (app 33.292s). `repoctl check` passed unit/vet, then
  failed docs-check because the supplied Japanese plan lacked `translation_of`
  and `source_sha256`; metadata is corrected in this documentation milestone.
- [x] Inspect final process runtime and Android UI observer boundaries.
- [x] Write English/Japanese Browser/CDP product/design docs.
- [x] Finalize browser binding syntax.
- [x] Define tested browser matrix.
- [x] Select minimal Go CDP/WebSocket implementation.
- [x] Implement browser capabilities and identity validation.
- [x] Validate private profile and exact CDP port binding.
- [x] Implement browser-level CDP connection.
- [x] Implement page/target model.
- [x] Implement Accessibility semantic snapshot.
- [x] Implement bounded DOM/layout snapshot.
- [x] Implement screenshot.
- [x] Implement stale-safe click/text/key/scroll.
- [x] Implement navigation and waits.
- [x] Implement console diagnostics.
- [x] Implement bounded network capture.
- [x] Add fencing, cross-lease and port-reuse regressions.
- [x] Add privacy/redaction tests.
- [x] Add iframe/shadow-DOM/multi-page fixtures.
- [x] Run the real Linux native fixture, including the lease-hosted backend.
  Chrome 152.0.7977.64, CDP 1.3, amd64, sandbox enabled: initial 7.072s pass,
  three subsequent repetitions passed, latest race run passed (package 8.489s,
  native test 7.47s). Windows/macOS subsequently passed as recorded below and in B25/B26.
- [x] Run real macOS browser integration: `9b94b42`, run 34234714187,
  darwin/arm64, Chrome 152.0.7977.82 / CDP 1.3, native test 20.98s PASS.
- [x] Run real Windows browser integration: `9b94b42`, run 34234714187,
  windows/amd64, Chrome 152.0.7977.82 / CDP 1.3, native test 25.54s PASS.
- [x] Run browser + lease-hosted backend E2E.
- [x] Update and validate durable documentation in English and Japanese: product,
  design, architecture, security/reliability, portability, quality, manifest/CLI,
  distribution, indexes and roadmap. `repoctl docs-check` passed at this checkpoint;
  final acceptance evidence will be reconciled after native CI.
- [x] Update standalone prerequisite matrix.
- [x] (2026-09-08) Run final harness/race/native/cross-build suites: Verify
  34235476057 and Browser native 34235476126 passed at `b48ab64`.
- [x] (2026-09-08) Complete B1–B34 direct evidence and bilingual retrospective.
- [x] (2026-09-08) Move both plans to `docs/exec-plans/completed/` and update
  references, reviewed translation hashes and documentation checks.

A checked item means observed completion. Record UTC date, revision, exact
browser version, protocol version, command/test/workflow and result.

### 2026-09-08 pre-push implementation checkpoint

- `go run ./tools/repoctl check` passed: formatting, unit tests, vet, docs,
  generation and architecture gates (app 8.352s, CLI 5.357s).
- `go test -race ./...` passed (app 37.289s, SQLite 15.069s). The latest focused
  app browser race suite also passed (2.855s), covering lost-fence output erasure,
  preserved identity fields, later entered-text redaction and tamper refusal,
  degraded-state guards and destroy fencing. Expanded CDP adapter tests passed.
- `TestBrowserNativeCLI` passed on Linux amd64 with sandbox enabled,
  Chrome 152.0.7977.64 / protocol 1.3: initial run, three repetitions and latest
  race run (8.489s package, 7.47s test). It checks empty input clearing, browser-name
  text without identity corruption, duplicate/replaced-node rejection without a
  backend effect, later-console Unicode redaction, and the complete fixture below.
- Six `CGO_ENABLED=0` CLI cross-builds passed: Windows/Darwin/Linux × amd64/arm64.
  These are compilation evidence only. Native Windows/macOS browser CI has not run.
- `TestArchitectureBoundaries` includes browser adapter dependency negatives;
  arch-check passed. Final docs-check is rerun after this evidence update.
- Full Docker `repoctl test-integration` subsequently passed. Explicit opt-in
  prerequisite tests retain their designed skips; this run does not claim optional
  native Android/Podman/browser gates. B25/B26/B33/B34 remain open.


### Final local implementation checkpoint (2026-09-08)

The final same-document URL digest, allowlisted AX states, Windows mixed-separator
profile validation and redacted-label matching refinements are implemented.
`go run ./tools/repoctl check` PASS after these refinements, including architecture
negative fixtures and translation checks. The final real Linux native race fixture
PASS (8.819s package / 7.81s test), Chrome 152.0.7977.64 / CDP 1.3, sandbox enabled.
Earlier pending local refinement checks are closed by this result. Windows/macOS
browser execution and published final CI remain pending; this plan stays active.

## Surprises & Discoveries

- 2026-09-09 — At `3d3fce5`, push Browser native 34246852839 passed all three
  OSes, but PR Browser native 34246856039 failed Windows during final profile
  deletion (`Cache_Data/sqldb0`: sharing violation). All browser operations and
  the new sandbox-access log guard had passed. Process Destroy already confirmed
  native tree absence and validated owned paths before its one RemoveAll call.
  The log does not identify the remaining filesystem holder, so do not attribute
  it to a specific process or kernel component. This is a cleanup acceptance
  failure despite the other native run passing; keep the plan active.

- 2026-09-09 — Real inherited-origin proof then exposed a sandboxed `srcdoc`
  iframe absent from `Page.getFrameTree` but present as an iframe target with
  selected-page parent IDs. Returning a root-only snapshot would falsely present
  complete observation. A target census now rejects related OOPIFs (and iframe
  targets whose ownership cannot be established), without inspecting their content.
- Another real navigation showed an empty child URL with unknown origin. Unlike
  committed opaque frames, this can be retried by wait under the existing deadline.
  Independent review also found a collection-time race: origins checked only before
  AX retrieval could label a navigated document with old frame identity. Final
  frame/tree/origin and target consistency checks must discard all evidence if
  collection crosses that change; the deterministic regression simulates it.

- 2026-09-09 — PR #10 regression tests failed before repair: URL waits accepted
  a role-only predicate; semantic input persisted no page/snapshot/node provenance;
  `gone` succeeded when AX bounds omitted the target; DOM names were shortened
  without reporting truncation; network metadata allowed 1,230,500 retained string
  bytes against the 65,536-byte budget; unsolicited or unrelated-session events
  could exhaust the 512-event queue and disconnect ordinary commands.
- Strict CDP security-origin classification exposed an actual browser protocol
  wrinkle: a same-origin inherited `about:blank` reports `securityOrigin="://"`
  even after load and with a loader ID. Three new native repetitions failed, so
  a protocol-mock pass was insufficient. Origin proof for these inherited frames
  requires an additional browser-enforced same-origin check, not a URL exemption.
- The recorded Windows failure includes explicit LPAC access denial for the
  installed CfT executable on both leases, followed by network-service crashes.
  Chromium's sandbox documentation requires installer/manual ACL setup; its own
  test helper grants the installation tree read/execute access for SID S-1-15-2-2.
  No frame-origin data was retained in that failed run, so the precise cause of
  its later observation refusal remains unproven until fresh native validation.

- Verify 34234714195's Linux integration/race job failed while constructing the
  `TestBrowserLifecycleGuards/dead` fixture: `sql: transaction has already been
  committed or rolled back`. This occurred before the browser assertion and is
  being investigated separately from Linux Chrome startup. Do not infer successful
  final harness acceptance from the native Windows/macOS passes.

- 2026-09-08 — Native CI [34234714187](https://github.com/mahcialet/agent-env/actions/runs/34234714187)
  confirmed macOS and Windows acceptance after `9b94b42`. Linux startup diagnostics
  revealed Chrome's fatal `No usable sandbox!` error, consistent with Ubuntu's
  AppArmor user-namespace restriction on unpacked Chrome for Testing. The initial
  readiness error alone had not exposed that host prerequisite.
- Verify [34233867022](https://github.com/mahcialet/agent-env/actions/runs/34233867022),
  retry of only the failed unchanged macOS readiness job at `506ed32`, passed.
  No code, timeout, or assertion changed for the retry. Full local race also passed
  after the CI repair (CLI 4.798s); independent review of `9b94b42` found no
  additional defects. Final revision CI remains required.

- 2026-09-08 — First published CI at `506ed3286e66fe0602c3d68189cbca5a13164dc6`
  failed native browser acceptance on all three platforms
  ([run 34233867023](https://github.com/mahcialet/agent-env/actions/runs/34233867023)).
  macOS reached Chrome 152.0.7977.82 / CDP 1.3 but empty text replacement failed
  equality readback; Windows reached the same version but the persisted-console
  privacy assertion failed; Linux failed the browser HTTP readiness probe. These
  failures are under investigation, not accepted platform evidence.
- The first Verify run also failed unchanged `TestHTTPReadinessAndObservedHealth`
  on macOS / Go 1.26.7: its successful-response phase exceeded the existing 20 ms
  probe timeout. Neither readiness implementation nor that test differs from the
  starting revision. Thirty local race repetitions passed; this does not establish
  a native macOS pass. Keep the failure visible and require successful final CI.

- Independent review found P1: after a lost lease fence, raw provider observation
  could remain in the error result before redaction. The app now clears observation
  output on this path; `TestBrowserLockLossDoesNotExposeObservation` verifies the
  output contains no raw secret. Lost-fence state is not finalized by the stale owner.
- Independent review found P2: redacting arbitrary structured strings could corrupt
  authority when entered text matched a browser name such as `web`. Redaction now
  preserves authority fields while redacting human content;
  `TestBrowserPriorTextRedactionPreservesAuthority` and real native text entry
  verify later snapshot reuse and redaction-proof tamper refusal.
- An intermediate integrated harness failed format-check while CDP transport was
  still being edited. Formatting was corrected and the complete harness passed;
  no check, test or portability requirement was weakened.

- Baseline unit/vet and full race checks passed, but the initial harness failed at
  docs-check: the supplied Japanese active plan lacked `translation_of` and
  `source_sha256`. This is a documentation input defect, not a passing baseline.
  The paired plan was meaning-reviewed and its metadata added in this milestone.
- Discovery alone cannot tie a loopback CDP listener to the native lease root.
  Connection validation therefore combines process-provider reinspection,
  `SystemInfo.getProcessInfo`, `Browser.getBrowserCommandLine`, discovery and
  `Browser.getVersion`. A forking launcher with a different browser PID is not a
  supported prerequisite; use the browser executable directly.
- DOM snapshots can contain input secrets in strings and attributes unrelated to
  the target input node. Retain structure/layout only, suppress editable AX values
  and retain durable fingerprint-only redaction evidence for entered text.
- Same-origin iframe and shadow observation is supported. Iframe semantic input
  is explicitly unsupported in this first slice; cross-origin frames/OOPIF do not
  become silent observation or input fallbacks.

Record Chrome/Chromium protocol differences, AX/DOM mismatches, OOPIF/iframe and
shadow-DOM behavior, target replacement during navigation, stale-node races,
console-history limits, network late-attachment gaps, profile-lock behavior,
browser auto-update differences, WebSocket teardown behavior and native OS
executable differences.

Do not weaken identity or stale-reference checks to make dynamic pages easier.

## Decision Log

- 2026-09-09 — Handle transient Windows sharing violations inside generic process
  state cleanup, only after existing native absence proof. Retry only that OS
  error, at most two seconds and bounded by caller cancellation; revalidate owned
  paths before every attempt. Other errors and persistent sharing violations
  remain failures with normal retention/quarantine behavior. Do not add Chrome
  lifecycle logic, ignore errors, lengthen process-stop grace, or release resources
  before the existing ownership and death checks pass.

- 2026-09-09 — Prefer CDP SecurityOrigin for frame classification. For inherited
  `about:blank`/`about:srcdoc` with Chrome's opaque placeholder, prove access using
  the native `contentDocument` getter in a verified parent's isolated world, with
  `grantUniveralAccess: false` (the CDP parameter spelling). Page-realm getter
  overrides cannot authorize access. A 512-target census rejects selected-page
  OOPIFs and unprovable iframe ownership. No external target is attached or adopted.
  Frame collection is checked again before publishing; changed or unavailable
  identity cannot produce partial evidence. Only wait may retry a transient
  uncommitted-origin/document-change error within its existing deadline.

- 2026-09-09 — Persist semantic page ID, source snapshot run ID and node reference
  in CommandRun before input. Preserve them for uncertain results and run.json,
  but never persist set-text content. Regression providers inspect the store
  during input, and successful/uncertain click, set-text, key and scroll results
  must retain the same provenance. URL waits separately require a nonempty
  substring and reject role-only or role-plus-substring requests before attachment.
- Capture queues subscribe to exactly one operation's session and supported
  event methods before enabling its domain; unsubscribe on all exits. Irrelevant
  notifications do not consume capacity, but subscribed overflow still fails
  closed. All console/network strings, including metadata, share the 4 KiB
  per-string and 64 KiB aggregate budgets; whole-field truncation avoids secret
  prefixes. Disappearance cannot be proven by an incomplete AX observation.
- Provision Windows CfT's own installation subtree with read/execute only for
  the restricted application-package SID, following
  [Chromium sandbox guidance](https://chromium.googlesource.com/chromium/src/+/main/docs/design/sandbox.md)
  and `testing/scripts/common.py`'s `set_lpac_acls`. Do not disable sandboxing or
  grant access to user/profile directories or unrelated parent paths. Native CI
  must validate this runner-only change; local Linux cannot validate Windows ACLs.

- 2026-09-08 — Limit the browser fixture's inherited 50 ms readiness deadline
  to the operations whose behavior it tests: permit five seconds only during
  fixture `Create`, then restore the original setting before every browser
  assertion. `waitReady` persists the lease with its readiness context, and an
  asynchronous SQLite rollback on deadline expiry can surface as `ErrTxDone`.
  CI failed during fixture construction, before marking the process dead.
  This supports a setup-deadline explanation; exact scheduler timing was not
  captured. Ten unmodified local race repetitions passed (20.980s), so it was
  not locally reproduced. Production readiness/locks and dedicated timeout,
  death, contention, cancellation, and fence-loss assertions are unchanged.

- 2026-09-08 — Provision an exact-path AppArmor profile for the pinned downloaded
  Chrome executable in the disposable Ubuntu CI runner, using Chromium's documented
  user-namespace allowance. This enables Chrome's sandbox while retaining the
  global restriction. Reject global sysctl relaxation and `--no-sandbox`. Host
  provisioning remains isolated to the Ubuntu workflow; core Go behavior and
  Windows/macOS execution are unchanged.

- 2026-09-08 — Send CDP's explicit `selectAll` editing command on the private
  selection keydown, retaining native Input events and transient equality readback.
  A platform accelerator alone did not reliably select nonempty text on macOS.
  Protocol regression covers Linux/macOS/Windows keydown and keyup semantics.
- Windows native process birth proof intentionally includes a Unicode guardian
  path. A broad `日本語` substring assertion confused that trusted path with entered
  text. Use a distinct input prefix and scan decoded JSON string values for the
  actual entered Unicode string (including quotes/backslashes), with regression
  coverage proving escaped secrets are detected and legitimate proof paths accepted.
  Production redaction and its acceptance requirement remain unchanged.
- Add failed-native-test process diagnostics before lease cleanup to obtain Linux
  Chrome startup evidence. Do not disable the sandbox or change host policy based
  only on a readiness timeout.

CI repair checkpoint: adapter race tests passed (1.415 s); real Linux browser
acceptance after the selection change passed (8.762 s), and strengthened privacy
checks plus failure diagnostics passed (8.243 s). Native macOS/Windows and Linux
CI remain pending. The failed unchanged macOS readiness test is being rerun on
the same published commit; no timeout or readiness assertion has been relaxed.

- 2026-09-08 — Implementation: adopt `browsers.<name>` with `type: chromium-cdp`,
  `runtime` and `cdp_port`. Each binding owns one existing process runtime's named
  TCP port; one binding per runtime. Explicit declaration prevents accidental
  browser semantics for unrelated processes. Required singleton argv switches are
  exactly `--headless=new`, `--enable-automation`,
  `--user-data-dir=${runtime_dir}/profile`,
  `--remote-debugging-address=127.0.0.1` and
  `--remote-debugging-port=${port:<cdp_port>}`. Alternate protected spellings,
  duplicates and profile/debugging overrides are rejected; no adapter appending.
- 2026-09-08 — Implementation: use `github.com/gorilla/websocket` v1.5.3 for an
  attaching CDP transport, without a browser launcher dependency or Node/Python.
  Domain defines browser values; app owns fencing/evidence; `internal/browser`
  owns protocol semantics, never process lifecycle. Native identity checks remain
  delegated to the existing process provider.
- 2026-09-08 — Implementation: require directly executable headless Chromium with
  exact native browser-root PID and command-line proof, not loopback discovery
  alone. Native matrix selection is Chrome for Testing 152.0.7977.82, Go 1.27 on
  Windows/macOS/Linux. Locally available Chrome 152.0.7977.64 protocol 1.3 is also
  exercised, without documenting its machine-specific path.
- 2026-09-08 — Implementation: all operations hold the lease fence. Degraded
  leases permit read-only diagnostics only with proven identity; quarantined,
  unfinished-run and ambiguous ownership states are refused. Mutating operations
  persist intent before input and retain a running cleanup barrier on disconnect
  or unconfirmed finalization; never replay input automatically.
- 2026-09-08 — Implementation: stale validation combines native/browser/page
  identity, document loader and backend DOM/frame fingerprint. Same-origin iframe
  and shadow DOM are observable, but iframe input and cross-origin/OOPIF handling
  are explicitly unsupported where identity/action support is unavailable.
  Fixed private target-bound JavaScript readback is allowed; arbitrary public
  JavaScript and CDP passthrough are not.
- 2026-09-08 — Implementation: network evidence stores no headers or bodies;
  URLs redact user information and query values. Console/network are bounded
  attachment-window captures. DOM stores structure/layout without text or
  attributes; AX suppresses editable/password values. Set-text records registered
  length/full-and-prefix SHA fingerprints before input, then later observations
  load that proof to redact echoed input. Missing/corrupt proof fails closed;
  fingerprint verification has a CPU budget. Proof files remain private and are
  not encryption. PNG pixels can contain secrets and are not automatically
  redacted.
- 2026-09-08 — Implementation: add explicit `page-create`, `page-close` and
  `dom-snapshot` operations to the minimum command surface. Use existing command
  run and artifact persistence; no new browser lifecycle table. No Playwright,
  download, external attachment or shared Android/browser abstraction is added.

- Decision: Browser/CDP is layered on PR #9 persistent process runtime.
  Rationale: process identity, private state, port allocation, logs and cleanup
  already belong to the generic runtime.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Initial scope is Chromium-family CDP only.
  Rationale: CDP provides Accessibility, DOMSnapshot, Page, Input, Runtime/Log
  and Network primitives required by the first browser contract.
  Date/Author: 2026-09-08 / maintainers.

- Decision: A private lease-owned user-data-dir is mandatory.
  Rationale: automation must never attach to personal browser state; modern
  Chrome also requires a non-default profile for normal remote debugging.
  Date/Author: 2026-09-08 / maintainers.

- Decision: External existing-browser attachment is out of scope.
  Rationale: CDP input requires proven process/profile/port ownership.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Accessibility tree is the primary agent-facing semantic snapshot;
  DOMSnapshot is complementary structure/layout evidence.
  Rationale: accessible role/name/value is better for agent actions while DOM
  evidence supports layout and debugging.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Node handles are ephemeral and snapshot-scoped.
  Rationale: navigation and dynamic page updates invalidate CDP DOM/AX identity.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Stale/ambiguous actions fail before input and never fall back to old
  coordinates.
  Rationale: stale observations must not operate another element.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Browser cleanup is delegated to persistent-process destroy, not
  `Browser.close`.
  Rationale: native process-tree and profile ownership are process-runtime
  responsibilities.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Chrome for Testing is preferred for CI but not bundled.
  Rationale: versionable automation evidence without enlarging standalone core.
  Date/Author: 2026-09-08 / maintainers.

- Decision: No arbitrary public raw-CDP or unrestricted JavaScript escape hatch
  in the first slice.
  Rationale: typed primitives keep behavior, safety and evidence reviewable.
  Date/Author: 2026-09-08 / maintainers.

- Decision: Durable docs and this ExecPlan are bilingual.
  Rationale: repository documentation policy.
  Date/Author: 2026-09-08 / maintainers.

- 2026-09-08 — Implementation: browser omission requires exactly one allocated
  binding; page omission requires exactly one eligible page, sorted by target ID.
  A page-close requires its explicit ID. AX nodes without backend identity remain
  observable but cannot authorize input. Shadow AX nodes use the same normalized
  representation. Unsupported methods fail explicitly rather than relying on a
  numeric protocol allowlist; discovery and live version observations must match.
  Each CDP call is bounded to five seconds within the operation deadline.
  Discovery is 64 KiB, WebSocket messages 8 MiB, pages 128, frames 32, AX/DOM nodes
  2048 and semantic JSON 1 MiB. Console/network retain 256 records and 64 KiB of
  strings each, with 4096 bytes per string; oversized strings become whole
  `[TRUNCATED]` markers. A 512-event transport buffer overflow fails capture.
  Console ignores pre-attachment replayed events. These bounds and unsupported
  cases are explicit so partial evidence cannot silently authorize an action.
  Downloads and headful mode are deferred; no shared Android/browser UI layer is
  justified in this slice.

- 2026-09-08 — Implementation refinement: any explicit navigate invalidates prior
  browser snapshots, even when the browser retains a loader during hash/history
  navigation. `TestBrowserNavigateInvalidatesSameDocumentSnapshot` is added for
  this policy (final rerun pending). The document token also includes a digest of
  the raw URL, never the raw query, to detect same-loader URL changes. Node
  fingerprints include allowlisted nontext AX state flags (`checked`, `selected`,
  `expanded`, `readonly`, `required`, `focusable`, `focused`, `multiselectable`).
  These changes avoid accepting semantically changed nodes/documents as fresh.
- 2026-09-08 — CLI refinement: require explicit `--text` for set-text while
  allowing an explicit empty string. Require meaningful key/URL/page/snapshot/node
  arguments and reject explicit zero durations before opening the store. This
  prevents omission from silently becoming a destructive empty text replacement.
  These latest refinements require final revalidation before acceptance.

## Outcomes & Retrospective

PR #10 review completed on 2026-09-09. Commit `3d3fce5` repairs origin proof,
wait predicates, truncation/byte budgets, event subscriptions and durable input
target provenance; `391288c` adds bounded Windows sharing-violation cleanup after
native absence proof. Both preserve ownership, sandboxing, privacy and the
process/CDP responsibility boundary. Regression tests first reproduced the defects;
real browser tests exposed inherited-origin placeholders and omitted OOPIFs that
mocks had missed. Independent review added post-collection identity checks so a
navigation cannot publish partial evidence under an old origin/loader.

Final revision `391288c351dec3e41febcfec15d912c65905a3f0` passed
[PR Verify 34247636419](https://github.com/mahcialet/agent-env/actions/runs/34247636419)
(all 12 jobs), [PR Browser native 34247636411](https://github.com/mahcialet/agent-env/actions/runs/34247636411)
(all three OSes) and [Release preview 34247636491](https://github.com/mahcialet/agent-env/actions/runs/34247636491)
(build and all three native smoke jobs). Push Verify/native also passed. All nine
threads received concrete replies and were resolved. This final archive changes
only documentation; the failed Windows runs and failed implementation approaches
remain below as historical evidence. The initial milestone retrospective follows.

Completed on 2026-09-08 on `feat/browser-cdp-automation`. This slice delivers
explicit Chromium-CDP bindings above the existing persistent process runtime.
Process ownership remains with that runtime; the adapter handles bounded CDP
transport, pages, AX/DOM observation, PNG screenshots, typed input and bounded
console/network capture. Live native PID/command-line/profile proof and the lease
operation fence precede effects. Registered snapshots bind lease, browser, page,
document and node identity; stale or ambiguous references fail without coordinate
fallback or replay. Set-text supports Unicode and explicit clearing with transient
readback, and durable fingerprints redact entered text across later CLI processes.

Final implementation/test revision `b48ab643a3e01029d880122b3c7c6830d82ed325`
passed [Verify 34235476057](https://github.com/mahcialet/agent-env/actions/runs/34235476057)
(all 12 jobs, including full race and integration) and
[Browser native 34235476126](https://github.com/mahcialet/agent-env/actions/runs/34235476126)
(all three OSes). Native Chrome 152.0.7977.82 / CDP 1.3 passed on Linux/amd64,
Windows/amd64 and macOS/arm64. Local harness, full race, real Docker integration,
six target builds and repeated sandboxed Linux browser tests also passed; the
acceptance table and dated checkpoints give direct evidence and its limits.

Independent review caught an early unredacted fence-loss return and authority
corruption from generic JSON redaction; both received targeted fixes and
regressions. Native CI then exposed macOS selection behavior, a Windows Unicode
proof-path false positive and Ubuntu's sandbox prerequisite. Explicit CDP editing,
decoded secret assertions and narrowly scoped runner provisioning resolved them.
A browser-fixture setup deadline was isolated from tested operation deadlines.
These discoveries show why protocol mocks and local success cannot replace native
acceptance. Earlier failed checks and the unchanged readiness-test retry remain
recorded. No production timeout, fence, sandbox or privacy requirement was relaxed.

Limits remain explicit: same-origin iframe and shadow observation are supported,
iframe input and cross-origin/OOPIF observation are not. No external attachment,
public script/CDP passthrough, download handling, automatic browser restart or
shared Android/browser UI layer was introduced. Network headers/bodies and DOM
editable values are excluded, but screenshots, unknown page text and the private
profile may contain sensitive data; this is not an encryption or browser-sandbox
boundary. The bilingual plan is archived with current documentation links updated.

## Context and Orientation

Read before implementation:

- `AGENTS.md` / `AGENTS.ja.md`
- `ARCHITECTURE.md` / `ARCHITECTURE.ja.md`
- `docs/PLANS.md` / `docs/PLANS.ja.md`
- merged PR #9 persistent-process product/design docs and completed plans
- Android UI observer product/design docs and completed plan
- `docs/PORTABILITY.md` / `.ja.md`
- `docs/SECURITY.md` / `.ja.md`
- `docs/RELIABILITY.md` / `.ja.md`
- `docs/QUALITY.md` / `.ja.md`
- `docs/roadmap.md` / `.ja.md`
- standalone distribution docs
- `internal/runtime/process`
- `internal/execx`
- `internal/app`
- `internal/domain`
- `internal/config`
- endpoint/readiness/evidence/store packages

Protocol references:
- https://chromedevtools.github.io/devtools-protocol/
- Accessibility
- DOMSnapshot
- Page
- Target
- Input
- Runtime/Log
- Network
- Chrome remote-debugging security guidance
- Chrome for Testing documentation

Repository docs remain the authority for agent-env behavior.

## Plan of Work

### Milestone 1 — Browser contract and binding

Create:

    docs/product-specs/browser-cdp-automation.md
    docs/product-specs/browser-cdp-automation.ja.md
    docs/design-docs/browser-cdp-automation.md
    docs/design-docs/browser-cdp-automation.ja.md

Preferred browser declaration is a top-level `browsers` collection bound to a
process runtime and named TCP port. An equivalent explicit model is acceptable if
it keeps browser semantics out of generic process config.

Validate before browser actions:

- referenced runtime exists and is `process`;
- named CDP port exists and is TCP;
- persisted expanded argv uses that exact CDP port;
- CDP address is loopback;
- `--user-data-dir` resolves inside the runtime's private state directory.

Do not introduce a new browser-owned process lifecycle.

### Milestone 2 — CDP transport and capabilities

Select a Go CDP implementation that attaches to an already-running browser.

Requirements:

- browser-level WebSocket;
- target/session support;
- bounded request deadlines;
- cancellation;
- event demultiplexing;
- no Node/Python helper;
- no browser process ownership.

Capabilities is read-only and reports browser product/version, protocol version,
runtime/page availability and implemented features.

### Milestone 3 — Browser and page identity

Before every operation:

1. resolve lease/browser/process runtime;
2. prove persistent process identity;
3. prove recorded CDP named port;
4. query loopback discovery;
5. connect to browser WebSocket;
6. verify product/protocol identity;
7. enumerate exact target/page.

Never connect to a different process that later reuses the old port.

Page handles are ephemeral; multiple pages require explicit selection unless
there is exactly one deterministic eligible page.

### Milestone 4 — Semantic and DOM snapshots

Primary snapshot uses CDP Accessibility.

Compact form:

    Snapshot br-...
    Page p1  http://...  "Sign in"

    [n1] heading "Sign in"
    [n2] textbox "Email" enabled
    [n3] textbox "Password" enabled password
    [n4] button "Login" enabled

Structured evidence retains frame, role/name/value, ignored state, backend DOM
identity when available, attributes/states and layout evidence required for
re-resolution.

DOMSnapshot is bounded complementary evidence. Do not dump unbounded full DOM in
normal output.

Node/byte limits and deterministic truncation markers are required.

### Milestone 5 — Screenshot

Use `Page.captureScreenshot` or equivalent.

Persist exact lease/browser/page identity, URL/title, dimensions, timestamp and
PNG digest. Invalid/incomplete base64 is failure.

Document that screenshots can contain secrets and pixels are not automatically
redacted.

### Milestone 6 — Stale-safe semantic input

Initial actions:

- click;
- set/replace text;
- key press;
- scroll.

Every semantic action references a prior snapshot/node.

Revalidate browser process, CDP endpoint, page and node before input. Prefer CDP
Input user-like events. Unicode text replacement needs a real native test.

No stored-coordinate fallback.

CDP disconnect during input is uncertainty; do not automatically replay.

### Milestone 7 — Navigation and waits

Implement explicit URL navigation and bounded waits for:

- lifecycle/load;
- URL;
- accessible role/name/text;
- node disappearance;
- optionally stable snapshot state.

Navigation invalidates prior snapshot handles. Timeout ends polling.

### Milestone 8 — Console and network diagnostics

Console evidence is bounded, attributed and redacted. Document whether messages
from before attachment can be recovered.

Network capture is explicit and bounded by duration/action. Capture
URL/method/status/type/timing/failure, redact Authorization/Cookie/Set-Cookie and
credential-like headers, and do not save response bodies by default.

No persistent unbounded background trace in this slice.

### Milestone 9 — Isolation and failure handling

Prove:

- snapshot from lease A cannot act on B;
- browser/page A snapshot cannot act on another browser/page;
- manual browser process death blocks CDP actions;
- unrelated port reuse is rejected;
- destroy cannot race through in-flight mutating browser action;
- profile deletion waits for proven process-tree absence;
- normal user profile is never used;
- no browser auto-restart.

Define DEGRADED read-only behavior before implementation. QUARANTINED ambiguous
identity never permits browser input.

### Milestone 10 — Real native integration

Build a deterministic local web fixture containing:

- heading/text;
- email/Unicode/password fields;
- button;
- DOM replacement;
- duplicate-label ambiguity case;
- iframe;
- shadow DOM;
- scrollable area;
- console output;
- network request;
- navigation.

Run a lease-owned headless Chromium-family browser on Windows/macOS/Linux.

Prove: discovery, pages, navigation, AX snapshot, DOM snapshot, screenshot,
Unicode input, semantic click, stale rejection, iframe/shadow observation,
console, network capture, later independent CLI observation, and safe destroy
with profile cleanup.

Prefer Chrome for Testing in CI and record exact versions.

### Milestone 11 — Browser plus lease-hosted application

Navigate the browser to an actual endpoint produced by agent-env:

- process runtime backend; and/or
- Compose/Podman backend available after merged work.

A semantic browser action should cause a backend request and produce correlated
browser/network/backend evidence.

Browser layer must not depend on one Compose provider.

### Milestone 12 — Documentation and completion

Update bilingual README, Architecture, Portability, Security, Reliability,
Quality, Roadmap, indexes and standalone prerequisite matrix.

Browser/CDP external prerequisite:

    compatible Chromium-family browser executable

Do not describe Chrome as bundled.

Complete acceptance evidence and retrospective before archiving both plans.

## Concrete Steps

1. Merge PR #9.
2. Fast-forward master and record revision.
3. Create branch.
4. Add bilingual active plans.
5. Run baseline harness/race.
6. Inspect process/UI observer boundaries.
7. Write bilingual product/design docs.
8. Finalize browser binding.
9. Implement CDP transport.
10. Implement capabilities/identity.
11. Implement page model.
12. Implement AX snapshot.
13. Implement DOM snapshot.
14. Implement screenshot.
15. Implement click/text/key/scroll.
16. Implement navigation/waits.
17. Implement console.
18. Implement network capture.
19. Add isolation/fencing/privacy/death/port-reuse regressions.
20. Add deterministic web fixture.
21. Run Linux real integration.
22. Run macOS real integration.
23. Run Windows real integration.
24. Add browser+backend E2E.
25. Update bilingual durable docs.
26. Run final harness/race/native/cross suites.
27. Record direct evidence.
28. Complete bilingual retrospective.
29. Move plans to completed and update links/hashes.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| B1 | Existing Compose/Podman/Android/Flutter/UI/process behavior remains valid. | Full local race/harness and Docker integration passed. Optional prerequisite-specific opt-in tests retain designed skips; no additional native Android/Podman pass is claimed. |
| B2 | Browser explicitly binds one owned process runtime and named TCP CDP port; no external-browser discovery. | `TestBrowserManifestContract`, `TestBrowserManifestNegativeFixtures`, `TestBrowserRequiresProcessRuntime`; `TestEndpointBoundary` rejects foreign endpoint authorities. |
| B3 | Managed browser always uses a private runtime-owned user-data-dir, never the user's default profile. | `TestPrivateProfileFlags` and config negative fixtures; Linux `TestBrowserNativeCLI` checks two distinct state directories, PIDs and CDP ports. |
| B4 | Every browser operation revalidates persistent process identity before CDP connection. | `TestBrowserLifecycleGuards` refuses dead/uncertain ownership before provider calls; `TestNodeChangesDuringOwnershipVerificationNeverInputs` verifies the action recheck. |
| B5 | Unrelated process port reuse cannot be mistaken for the owned browser. | `TestBrowserPIDAndDiscoveryProof` rejects wrong PID, version and endpoint; `TestEndpointBoundary` rejects foreign ports. These controlled transport negatives do not claim a real kernel port-reuse race. |
| B6 | Capabilities reports product/version/protocol with no navigation/input effects. | Linux `TestBrowserNativeCLI` records Chrome 152.0.7977.64 / CDP 1.3 and verifies the original page stays about:blank after capabilities. |
| B7 | Page enumeration is deterministic; multiple pages require unambiguous selection. | Linux `TestBrowserNativeCLI` creates/closes a second page and rejects implicit multi-page snapshot; adapter sorts page IDs. |
| B8 | Accessibility snapshot returns deterministic versioned JSON and compact semantic text. | `TestBrowserSnapshotRegistrationAndSemanticInput` verifies registered snapshot artifacts; native fixture checks semantic heading/button/textbox roles and iframe/shadow nodes. |
| B9 | DOM/layout evidence is bounded and explicitly truncated when needed. | `TestDOMSnapshotSuppressesSensitiveStrings`, `TestSnapshotOmitsOversizeSensitiveValue`; native DOM artifact validates version 1, nonempty nodes and ≤2048 nodes. |
| B10 | Screenshot yields a valid PNG digest tied to exact browser/page identity. | Native fixture decodes PNG, verifies positive dimensions, SHA256 digest and lease/run/browser/page attribution; `TestScreenshotMessageBoundAndDecodeFailure` covers malformed/oversized messages. |
| B11 | Fresh semantic click operates the intended revalidated node. | Native click yields backend count 1 and correlated process logs; replacing a node then using its old handle causes no second backend effect. |
| B12 | Unicode text replacement works without shell/JS-string escaping corruption. | Native fixture round-trips Japanese, emoji, accents, quotes/backslash through CDP and backend; verifies empty clearing and input equal to browser name. |
| B13 | Key and scroll actions affect only the selected page. | Native Enter and scroll produce the expected page text; independent second lease/page remains about:blank and its backend count remains zero. |
| B14 | Stale/ambiguous node fails before input and never falls back to stored coordinates. | `TestStaleAndAmbiguousNodeNeverInputs`, `TestNodeChangesDuringOwnershipVerificationNeverInputs`; native replacement and navigation invalidate prior handles. |
| B15 | Navigation is bounded and invalidates prior snapshot handles. | Native navigation to /next rejects prior-page snapshot input and satisfies bounded URL wait. |
| B16 | Waits terminate on success/timeout with no hidden continued polling. | `TestWaitUsesOperationDeadlineRatherThanCaptureDuration`; native missing-text wait uses 250ms timeout and checks it ends within a 5s outer allowance. |
| B17 | Console diagnostics are bounded, attributed, redacted and document history limits. | `TestConsoleIgnoresHistoryAndOmitsOversizeValues`; native heartbeat and later Unicode echo capture verify redaction in output and registered JSON. |
| B18 | Network capture is bounded and redacts auth/cookie headers by default. | Native network capture correlates /tick request IDs to status 200 and rejects credential/header test strings in artifacts. Model stores no headers or bodies; limits documented. |
| B19 | Cross-lease/browser/page snapshot/action reuse is rejected. | Native fixture rejects cross-lease and cross-page handles; app checks snapshot browser/page match and digest registration before provider input. |
| B20 | Destroy cannot race past an in-flight mutating browser operation. | `TestBrowserMutationFenceBlocksDestroy`, `TestBrowserUncertainMutationRetainsCleanupBarrier`, `TestBrowserEvidenceFailureRetainsBarrier`. |
| B21 | Manual browser process death causes later CDP action refusal. | Native fixture kills the second browser root and verifies subsequent browser pages refuses; show reports non-ready with unchanged historical PID. |
| B22 | Profile deletion occurs only after process-tree absence is proven; uncertainty retains/quarantines. | Native fixture destroys both leases and checks state/profile directories absent; generic `TestMissingLaunchingReceiptIsUncertain` and browser uncertain/evidence barriers preserve conservative cleanup. |
| B23 | Browser process lifecycle remains owned by persistent-process runtime; no duplicate PID cleanup. | `TestArchitectureBoundaries` browser dependency negatives and arch-check passed; native fixture cleanup calls ordinary destroy, not CDP Browser.close. |
| B24 | No automatic browser restart. | `TestBrowserLifecycleGuards` checks start count unchanged; native manual-death fixture keeps historical PID and never returns ready. |
| B25 | Native Windows | PASS: `391288c`, PR Browser native 34247636411, windows/amd64, Chrome 152.0.7977.82 / CDP 1.3, real CLI fixture 26.76s including sandbox-access guard and normal cleanup. |
| B26 | Native macOS | PASS: `391288c`, PR Browser native 34247636411, darwin/arm64, Chrome 152.0.7977.82 / CDP 1.3, real CLI fixture 9.60s. |
| B27 | Real native headless Browser/CDP integration passes on Linux. | PASS: local sandboxed Chrome 152.0.7977.64 / CDP 1.3, final race native package 8.834s; PR native 34247636411 at `391288c`, Chrome 152.0.7977.82 / CDP 1.3, linux/amd64 test 8.86s. |
| B28 | Real feature fixture | All three native OS fixtures passed at `391288c` (PR Browser native 34247636411), including inherited blank/srcdoc/blob observation and opaque/OOPIF refusal. Iframe input and cross-origin observation remain unsupported. |
| B29 | Browser exercises a lease-hosted backend without one Compose-provider dependency. | Native fixture builds a repository-owned HTTP backend as another process runtime, correlates Unicode request/count/log evidence, and proves the second lease backend unaffected; no Compose provider used. |
| B30 | Core standalone commands do not require a browser when Browser/CDP is unused. | Full core unit/race suites and six CGO-free CLI cross-builds passed without browser integration tag; browser prerequisite is required only by explicit browserintegration tests/commands. |
| B31 | No Node/Python/Playwright/Selenium/ChromeDriver runtime dependency is introduced. | Go gorilla/websocket transport and direct native argv; architecture checks passed, no helper runtime/bundled browser added. |
| B32 | Bilingual durable docs describe final behavior and privacy limits. | Paired product/design and supporting documents updated with exact flags, native prerequisites, privacy/limits; docs-check passed and hashes refreshed after meaning review. |
| B33 | Final checks | PASS: `391288c` PR Verify 34247636419 (12 jobs, full race/integration), PR Browser native 34247636411 (3 OSes), Release preview 34247636491 (build/native smoke), matching push CI, local harness and reviewed bilingual documentation. |
| B34 | Living plan/evidence | PASS: PR #10 review gates and B1–B34 reconciled; all nine threads replied/resolved; failures retained, bilingual outcomes updated and both plans re-archived after final acceptance. |

Code existence alone is not acceptance. Record exact browser/protocol versions,
native runs and observed behavior.

## Idempotence and Recovery

Read-only browser observation does not change desired lease state.

Mutating browser operations are not generally idempotent. Record intent/result,
make one input attempt, and never automatically replay uncertain actions. A fresh
snapshot/state check is required before retry.

A CDP disconnect during input does not prove input was absent.

Never recover by attaching to a different listener on the old port or to the
user's normal browser.

Profile contents are private mutable runtime state, not automatic evidence.

Destroy/GC remains delegated to persistent-process ownership proof.

## Artifacts and Notes

Suggested artifacts:

    leases/<id>/artifacts/<browser-run-id>/
      run.json
      snapshot.json
      snapshot.txt
      dom-snapshot.json
      screenshot.png
      redaction.json  (before set-text; fingerprint-only proof)

Not every operation creates every file. Console/network arrays are stored in
`run.json`, not separate collection files.

Record lease/browser/process identity, CDP port, product/version/protocol,
page/target, URL/title, snapshot/node, operation timestamps, truncation, artifact
digests and result/uncertainty.

Do not persist passwords or secret text in clear action metadata. DOM text,
console, URLs and screenshots may themselves contain secrets.

## Interfaces and Dependencies

Implemented adapter package:

    internal/browser/cdp/

Implemented app port:

```go
type BrowserProvider interface {
    Observe(context.Context, domain.Runtime, domain.BrowserBinding,
        domain.BrowserRequest, func(context.Context) error) (domain.BrowserObservation, error)
}
```

Responsibilities:

- process runtime: process/profile-dir/port lifecycle;
- app: lease selection, fencing, stale/action/evidence policy;
- browser/CDP adapter: protocol transport/parsing;
- store: persistence;
- CLI: parsing/rendering.

External runtime dependency only when Browser/CDP is used:

    compatible Chromium-family browser executable

No mandatory Node, Python, Playwright, Selenium, ChromeDriver, shell, CGO or
daemon dependency is added.

## Milestone 1 Decisions to Reconcile at Acceptance

The Decision Log resolves these initial questions; retain the checklist for
acceptance traceability.

1. Final browser binding location/name (`browsers` preferred).
2. Whether browser flags are repository-declared only or a binding may append a
   strictly controlled required set.
3. Initial tested browser matrix.
4. Go CDP/WebSocket library.
5. Protocol compatibility/capability strategy.
6. Exact process/profile/CDP identity proof.
7. Default page-selection rule.
8. AX stale fingerprint when backend DOM identity is absent.
9. OOPIF/cross-origin iframe handling.
10. Shadow DOM normalization.
11. Unicode text replacement without unrestricted JS.
12. Console history semantics.
13. Network event/byte limits.
14. Download handling.
15. Headless-only product contract versus headful support.
16. Whether Android UI and Browser snapshots/actions justify a later shared UI
    abstraction.

Resolve these in Decision Log before dependent behavior is declared stable.

2026-09-08 CI checkpoint: all three real-browser jobs passed at `a37f11f`
([Browser native 34235169459](https://github.com/mahcialet/agent-env/actions/runs/34235169459)),
including Ubuntu's scoped AppArmor allowance with sandbox and global restriction
still enabled. The browser-fixture-only readiness preparation change passed ten
local race repetitions of every `TestBrowser*` (20.222s). Full Verify is pending.

Final native evidence at `b48ab643a3e01029d880122b3c7c6830d82ed325`:
[Browser native 34235476126](https://github.com/mahcialet/agent-env/actions/runs/34235476126)
passed all three jobs with Chrome 152.0.7977.82 / CDP 1.3 and Go 1.27.
Linux/amd64 native fixture: 8.16s; Windows/amd64: 29.97s; macOS/arm64: 19.62s.
The JSON privacy-detector regression also passed on all three platforms. These
results supersede earlier native-pending checkpoints without erasing the failures.

Final closure (2026-09-08): Verify 34235476057 completed successfully at `b48ab64`
with all 12 jobs passing. Combined with Browser native 34235476126, this closes
all acceptance gates left open in earlier dated checkpoints. The final change
from that verified revision only reconciles bilingual documentation and archives
this plan; no runtime or test behavior changes are part of the archive milestone.

2026-09-09 review checkpoint: targeted app URL/provenance race passed (2.056s);
full app and CDP race passed (36.887s / 1.735s). Capture/transport regressions
passed ten race repetitions (4.392s). Independent review found no additional
defect in app/provenance, transport/capture or scoped Windows ACL provisioning.
Documentation checks passed. Origin proof/native acceptance is still being
implemented and tested; the plan stays active.

2026-09-09 integrated checkpoint before the final collection-race guard: full
`repoctl check` and `go test -race ./...` passed; actual sandboxed Linux Chrome
152.0.7977.64 / CDP 1.3 passed three strengthened native race repetitions (25.160s).
The real fixture covers inherited blank/srcdoc, same-origin blob and opaque OOPIF
refusal despite a malicious page-realm getter override. The final race guard and
Windows LPAC setup still require fresh validation before the acceptance gate closes.

2026-09-09 final local review repair: the post-collection consistency guard passed
full CDP race (2.331s) and the actual sandboxed Linux native fixture (8.457s).
Independent review verified closure of the collection-race finding and found no
additional defect. Final full harness/race passed after integration. The Windows
fixture now also rejects the previously observed executable sandbox access-denial
log. Fresh multi-OS CI and thread replies remain required before completion.

2026-09-09 cleanup checkpoint: process race passed (1.964s), targeted cleanup/
Destroy regressions passed ten race repetitions (2.916s), and Windows amd64/arm64
test binaries cross-compiled. A native Windows test holds an actual file without
delete sharing, verifies RemoveAll's sharing failure, then releases it to exercise
successful cleanup. Independent review found no new defect. The two-second
budget bounds retry scheduling, not a synchronous filesystem call in progress.
Final local harness and strengthened Linux native validation passed; Windows
execution and fresh full CI remain pending.

2026-09-09 final native/release evidence at
`391288c351dec3e41febcfec15d912c65905a3f0`: both
[push Browser native 34247632201](https://github.com/mahcialet/agent-env/actions/runs/34247632201)
and [PR Browser native 34247636411](https://github.com/mahcialet/agent-env/actions/runs/34247636411)
passed all three OSes, including Windows normal profile deletion and the
sandbox-access guard. PR Chrome 152.0.7977.82 / CDP 1.3 native times were
Linux/amd64 8.86s, Windows/amd64 26.76s and macOS/arm64 9.60s.
[Release preview 34247636491](https://github.com/mahcialet/agent-env/actions/runs/34247636491)
also passed build and native smoke jobs. Eight review threads are replied/resolved;
the plan gate remains open only until final Verify acceptance is confirmed.

Final PR #10 closure (2026-09-09): all review Progress and acceptance gates are
complete. Final Verify 34247636419 and push Verify 34247632059 passed at `391288c`.
The native and release runs above also passed. All nine threads are resolved.
This record supersedes the earlier pending checkpoints; source/test code is
unchanged by the bilingual archive milestone.
