---
status: active
plan_id: EP-QUAL-001
plan_type: implementation
priority: 20
merge_policy: guarded
base_branch: master
branch: feat/ep-qual-001
workstreams:
  - testing
  - repoctl
  - documentation
owner: maintainers
last_verified: 2026-09-11
---

# Audit test architecture and evidence governance

[日本語](test-architecture-evidence-governance.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Plan ID: `EP-QUAL-001`

Expected branch: `feat/ep-qual-001`

Starting revision: `cbd84ed1d2ef4456b8a95db215461bd3d726fe59`, the `master` merge of PR #14. Do not start implementation from the `EP-OPS-001` feature branch or from an unmerged stacked revision.

## Purpose / Big Picture

Audit the repository's tests as a system in their own right, repair test architecture defects, and strengthen agent instructions so that passing tests cannot be presented as stronger evidence than they actually provide.

The immediate trigger is a test intended to detect recurrence of a Browser/CDP defect. After client cancellation ended the client wait, the mock server callback could continue executing, but the test read callback-owned state without synchronization. The test was meant to detect an invalid ordering assumption in production code. Yet it relied on another such assumption: that the server callback had finished when the client wait ended.

This Plan therefore does not ask only:

    Is production behavior covered by tests?

It also asks:

    Is the test oracle valid?
    Is the failure schedule actually exercised?
    Is the fixture itself concurrency-safe?
    Does cleanup establish completion?
    Can the test pass without reaching the behavior it claims to test?
    What exactly does each reported validation result prove?

The target outcome is:

    production invariant
        -> explicit test oracle
        -> trustworthy fixture
        -> deterministic failure schedule where applicable
        -> fail-before evidence
        -> repair
        -> pass-after evidence
        -> correctly classified supporting evidence

Repeated success, CI reruns, race-detector passes and cross-platform passes remain useful observations, but their quantity must not upgrade them into proof of an invariant they do not force.

This Plan is also intended to be the first new ordinary ExecPlan used for forward live dogfooding of the lifecycle introduced by `EP-OPS-001`.

## Scope

In scope:

- all repository-owned Go tests and shared test helpers;
- concurrency-bearing fixtures and helpers;
- goroutine, process, socket, server and client lifecycle in tests;
- cancellation, deadlines, polling and cleanup;
- mutable state shared between test goroutines;
- timing-dependent tests, sleeps and small timeout assumptions;
- mock/fake behavior and fidelity;
- deterministic failure injection;
- positive and negative controls;
- error-only assertions and false-positive test paths;
- callback/effect reachability assertions;
- test oracle independence;
- boundary and composition coverage;
- fixture ownership and cleanup proof;
- race-sensitive tests;
- use of `-count=N`, CI reruns and repeated native execution as evidence;
- evidence classification and reporting rules;
- `AGENTS.md`, `QUALITY.md`, plan policy or other durable guidance needed to encode those rules;
- targeted mechanical checks where a stable invariant can be enforced without excessive false positives;
- confirmed repairs discovered by the audit;
- independent review of the audit and its repairs.

Initial high-priority areas:

- `internal/browser/cdp`;
- `internal/execx`;
- application cancellation and readiness tests;
- controller/client/worker and multi-host fixtures;
- process-runtime fixtures;
- Android Emulator / Android UI helper fixtures;
- Compose integration fixtures;
- release and filesystem mutation/failure-injection tests.

Out of scope:

- replacing all mocks with real integration tests;
- treating integration testing as universally stronger than deterministic unit testing;
- banning `time.Sleep`, deadlines, polling or atomics categorically;
- increasing timeouts merely to make failures disappear;
- converting every test to property-based testing;
- rewriting completed historical ExecPlans to make old evidence conform to the new terminology;
- claiming exhaustive exploration of scheduler interleavings;
- weakening existing production correctness, portability or native verification requirements;
- adding a general-purpose model checker unless the audit demonstrates a concrete need.

## Progress

- [x] Record PR #14 merge revision `cbd84ed1d2ef4456b8a95db215461bd3d726fe59` and create `feat/ep-qual-001`.
- [x] Run and record the untouched baseline repository harness, full race suite and relevant native checks.
- [x] Freeze the audit method and evidence taxonomy before repairing findings (kickoff record below).
- [x] Inventory repository-owned test helpers and concurrency-bearing fixtures.
- [x] Audit Browser/CDP test architecture, including `mockBrowser` and `eventBrowser`.
- [x] Audit cancellation/deadline tests across the repository.
- [x] Audit process, worker/controller, Android and integration fixture lifecycles.
- [x] Audit test oracles for false-positive paths and unreachable intended effects.
- [x] Classify all findings and record ACCEPT / REJECT / DEFER decisions with rationale.
- [x] Repair accepted test-architecture defects with fail-before evidence.
- [x] Add deterministic failure schedules for accepted concurrency/order findings where practical.
- [x] Add or improve lifecycle synchronization for affected helpers.
- [x] Define repository evidence classes and anti-evidence-laundering rules.
- [x] Update agent instructions and quality documentation.
- [x] Add targeted mechanical preventive controls where justified.
- [x] Independently review the audit corpus, dispositions and repairs; address worker-join, stage-wait and live-container-oracle findings (2026-09-10).
- [x] Run final focused deterministic regressions.
- [x] Run final full race/harness checks (2026-09-10).
- [x] Run required native Windows/macOS/Linux validation, including Q14 follow-ups (2026-09-10).
- [x] Complete bilingual reader and semantic parity review; resolve two JA omissions and record the withdrawn M5 finding (2026-09-10).
- [x] Clarify rejection/error-test terminology in both languages without changing recorded evidence (2026-09-11).
- [ ] Complete retrospective and archive only after merge into `master`.






## Surprises & Discoveries

- Baseline docs-check exposed stale draft metadata; the kickoff migrated it to the current schema.
- Windows guardian polling ignored every observation error, hiding the exact sharing violation propagated by the runtime caller. The handle owner is unknown; the fix only treats a sharing conflict on a verified empty Job as still pending.
- The old test that checks refusal to confirm node absence from a truncated snapshot passed an injected unrelated early failure. The strengthened oracle rejects the identical injection before counting the expected boundary as exercised.

Record findings that change the audit model, not only individual test defects.

In particular, record:

- tests that pass without reaching the intended mutation/effect;
- tests whose oracle depends on the implementation under test;
- test cleanup functions that close resources but do not wait for work to finish;
- callbacks that execute concurrently with assertions;
- atomics that remove a data race without establishing the ordering required by the test;
- timeout increases that hide scheduling assumptions;
- mock behavior that is materially different from the real protocol/runtime;
- repeated tests that never force the claimed failure schedule;
- native or CI passes that were previously described more strongly than their actual evidence permits;
- sibling fixtures that share the same defective assumption;
- places where a mechanical checker would create more noise than useful enforcement.

Do not repair a finding before recording enough evidence to explain the violated invariant and the earliest realistic prevention point.

## Decision Log

- 2026-09-10 / maintainer-approved kickoff: Promote the supplied draft to active, use deterministic branch `feat/ep-qual-001`, and remove the circular EP-OPS-001 completion dependency. The merged implementation is the foundation; outstanding Human Validation is not waived.

- 2026-09-10 / maintainers: Treat tests and fixtures as software requiring their own correctness review, not merely as evidence about production code.
- 2026-09-10 / maintainers: A passing race detector is evidence that an observed execution contained no reported data race; it is not proof that all relevant schedules were exercised.
- 2026-09-10 / maintainers: Repeated execution such as `-count=N` is stability/flakiness evidence only unless the test deterministically constructs the relevant state or ordering.
- 2026-09-10 / maintainers: Quantity does not upgrade evidence class. Many weak observations do not become deterministic proof.
- 2026-09-10 / maintainers: For concurrency defects, prefer an explicit happens-before relation or deterministic barrier over timing probability.
- 2026-09-10 / maintainers: Adding an atomic is insufficient when the semantic requirement is operation completion or ordering rather than merely race-free access.
- 2026-09-10 / maintainers: Timeout increases are not accepted as a concurrency repair unless the timeout itself is the violated product/test contract.
- 2026-09-10 / maintainers: Accepted findings should identify the violated invariant, escape path, sibling exposure and preventive control rather than stopping at the failing line.
- 2026-09-10 / maintainers: Preserve historical completed-plan evidence. Apply the new evidence vocabulary prospectively and clarify current authoritative documentation instead of rewriting history.
- 2026-09-10 / maintainers: Keep `AGENTS.md` navigational and concise; detailed evidence semantics belong in quality/design documentation and enforceable checks.

## Outcomes & Retrospective

Implementation through Q14 is validated locally and by native/integration CI.
Merge/archive acceptance remains open. The bounded audit covered 195 baseline test files across 28 package
directories and recorded Q01–Q14. Thirteen ledger entries were accepted and repaired;
Q09's unproven stream/constructor failure-path concern is explicitly deferred.
The corpus now has 199 test files, including owned fixture controls.

The audit demonstrated old-oracle false passes, added discriminating failure
stages and explicit lifecycle completion, and retained native evidence limits.
Native CI caught an EOF-only portability assumption in the first repair, and
race CI exposed an expiring one-shot host registration. Both were corrected
without weakening deadlines, controller TTLs or portability checks. Independent
review also rejected two scheduling assertions in new tests before their
acceptance. Passing local tests and source review were useful but insufficient
substitutes for these native and forced-fault controls.

Do not archive until PR #15 has current-HEAD human review and is merged into
master. The trusted-base gate policy is absent, so guarded/manual fallback
continues to apply; no automatic merge authorization is inferred.

Twelve accepted ledger entries are test-only defects; Q08 affects production
completion observation and its test oracle. CDP and app/process helpers had the
largest concentration of lifecycle and reachability gaps. Earlier repeat-pass
results did not prove callback completion or cross-platform socket semantics.
Tests of specific conditions that must remain true can be executable controls; evidence classification,
bounded sibling review and language meaning remain review disciplines. Q09,
unexposed inline client-reader joins, unforced schedules and environments not
executed remain explicit limits rather than claims of exhaustive correctness.

This new Plan exercised EP-OPS-001's deterministic branch, stable Plan ID,
commit/PR trailers, provenance and guarded gate inspection successfully through
implementation. The gate correctly retained manual fallback without trusted
policy; Human Validation was not auto-started. Forward merge/archive evidence
remains open until the maintainer finishes PR #15.

This section remains incomplete until the Plan has merged.

The final retrospective must answer:

- Which test-architecture defect classes were found?
- Which subsystems were most affected?
- How many findings were production gaps versus test-only defects?
- Which findings could previously produce false confidence?
- Which repeat-pass or CI evidence had been overinterpreted?
- Which helper contracts changed?
- Which rules became mechanically enforceable?
- Which rules remain review disciplines rather than static checks?
- Did the audit discover defects in production code while inspecting test assumptions?
- What classes of failure remain fundamentally probabilistic or unverified?
- Did this Plan successfully exercise the `EP-OPS-001` forward lifecycle on a real new work item?

Do not summarize success using test counts alone.

## Context and Orientation

Read before changing code:

- `AGENTS.md` / `AGENTS.ja.md`;
- `docs/PLANS.md` / `docs/PLANS.ja.md`;
- `docs/QUALITY.md` / `docs/QUALITY.ja.md`;
- repository-correctness audit matrix, findings and subsystem annexes;
- completed Browser/CDP ExecPlan;
- completed process-runtime and multi-host ExecPlans;
- `internal/browser/cdp/*_test.go`;
- shared test helpers in other runtime/application packages;
- `tools/repoctl`;
- GitHub Actions workflows.

The existing repository correctness audit already classifies defects using categories such as:

    CONCURRENCY_GAP
    COMPOSITION_GAP
    FAILURE_INJECTION_GAP
    ORACLE_COUPLING
    NEGATIVE_FIXTURE_GAP
    BOUNDARY_GAP

Reuse those where appropriate. Add test-architecture-specific categories only when they distinguish a materially different escape mechanism.

Candidate additions include:

    FIXTURE_LIFECYCLE_GAP
    NONDETERMINISTIC_ORACLE
    EVIDENCE_OVERCLAIM
    UNREACHED_EFFECT
    TIMING_PROXY
    CLEANUP_COMPLETION_GAP

Do not proliferate labels without a clear review use.

## Plan of Work

### Milestone 0 — Baseline and audit freeze

After `EP-OPS-001` is merged:

1. Record exact `master` SHA.
2. Create the expected branch.
3. Run the repository's normal checks without modification.
4. Run the full race suite.
5. Preserve any existing failure rather than immediately retrying it away.
6. Record current relevant CI/native status separately from local results.

Before changing any failing fixture, write the audit rubric and evidence taxonomy so that the repair is judged by rules established before observing the desired green result.

### Milestone 1 — Evidence taxonomy

Define evidence by what it demonstrates, not by how impressive the quantity appears.

At minimum distinguish:

**Deterministic invariant evidence**

The test intentionally forces the state, transition, ordering, boundary or failure being claimed.

Examples:

    explicit channel/barrier schedule
    fail-before -> repair -> pass-after
    exact boundary fixture
    injected persistence failure
    controlled ownership mismatch

**Direct native/integration evidence**

Real supported runtime behavior was observed in a declared environment.

Useful for platform/runtime semantics, but not automatically deterministic proof of rare schedules.

**Race/static/tooling evidence**

Examples:

    go test -race
    go vet
    architecture validator
    docs validator

Each tool proves only its own checked property.

**Stability evidence**

Examples:

    -count=50
    repeated CI
    multiple CPU settings
    repeated native runs

This can reveal flakiness and increase the chance of observing a probabilistic defect. It does not prove that an unforced schedule occurred.

**Compilation/structural evidence**

Examples:

    cross-build
    generated-file comparison
    schema validation

Never describe these as native execution evidence.

Add the following rule verbatim or equivalently to authoritative guidance:

    Evidence must be classified by what it proves.
    Quantity does not upgrade evidence class.
    Repetition is not a substitute for deterministic reproduction.

When stability evidence is reported for a concurrency bug, require an explicit statement of what schedule or invariant it does not prove.

### Milestone 2 — Test-helper and fixture inventory

Inventory helpers that own or indirectly start:

- goroutines;
- HTTP/WebSocket servers;
- clients/read loops;
- subprocesses;
- process trees;
- Docker/Podman resources;
- Android/ADB/Emulator processes;
- controller/worker services;
- temporary databases with concurrent users;
- files intentionally mutated during reads;
- timers/poll loops.

For each relevant helper record:

    owner
    resources started
    concurrency introduced
    completion signal
    cleanup behavior
    whether cleanup waits
    mutable shared state
    failure injection surface
    tests consuming it
    known assumptions

The inventory may live in a bounded audit document rather than permanent product documentation.

### Milestone 3 — Deterministic concurrency audit

Start with Browser/CDP.

For `mockBrowser`, determine and document the lifecycle of:

    request received
    callback entered
    callback completed
    response write attempted
    response observed or connection closed
    handler exited
    server shutdown completed

The helper must not imply that client cancellation establishes server callback completion unless it actually does.

For the load-wait regression, construct the relevant cancellation/server overlap deliberately rather than relying on scheduler timing.

The desired regression shape is conceptually:

    server enters evaluation
    -> test observes entry
    -> client is cancelled / deadline expires
    -> client operation returns
    -> test proves server work can still be active
    -> test releases server work
    -> server completion is observed
    -> final state is asserted

The implementation may use channels, barriers, WaitGroups or another clear synchronization primitive.

Do not add atomics as the sole repair if assertions semantically require completion.

Audit sibling `mockBrowser` cancellation/deadline users and `eventBrowser`.

Then apply the same lifecycle questions to other concurrency-bearing fixtures.

### Milestone 4 — Oracle and reachability audit

Search for tests that can succeed for the wrong reason.

Patterns include:

- assertion checks only `err != nil`;
- expected callback/effect is never asserted as reached;
- refusal happens earlier than the behavior the regression claims to cover;
- fake and implementation share the same incorrect assumption;
- positive and negative cases are coupled through one condition;
- boundary tests accidentally trigger multiple limits at once;
- cleanup assertions occur before cleanup completion;
- context cancellation is confused with subordinate work termination.

For mutation/effect tests, add an explicit reachability oracle where valuable:

    mutation callback executed
    destructive operation remained uncalled
    expected request arrived
    intended boundary was reached
    expected persistence step occurred

Do not mechanically add counters everywhere. Add them where they distinguish the intended path from a false-positive path.

### Milestone 5 — Finding ledger and disposition

For every confirmed finding record:

    ID
    subsystem / fixture
    observed defect
    violated invariant
    earliest realistic detection stage
    why existing tests/review missed it
    whether the issue affects production, tests, or both
    sibling exposure
    evidence needed for repair
    disposition: ACCEPT / REJECT / DEFER
    rationale
    preventive control

A local fix is not complete until sibling exposure has been examined.

REJECT and DEFER entries remain in the ledger with rationale.

### Milestone 6 — Repairs

Repair accepted findings in coherent slices.

For each concurrency/order finding:

1. Establish deterministic fail-before evidence where practical.
2. Repair the lifecycle/order contract.
3. Demonstrate pass-after using the same deterministic reproducer.
4. Run the race detector.
5. Run relevant sibling tests.
6. Use repetition only as supplementary stability evidence.
7. Record remaining unforced schedules explicitly.

Avoid broad refactors merely to make tests aesthetically uniform.

Prefer test helpers whose APIs expose the lifecycle a caller actually needs to reason about.

### Milestone 7 — Agent instruction and quality-policy changes

Update durable guidance so future agents do not optimize for green-looking proxy metrics.

`AGENTS.md` should remain concise. Add only the essential workflow rules, such as:

    Identify the violated invariant before choosing the repair.
    For concurrency failures, distinguish cancellation from completion.
    Do not present repetition as deterministic proof.
    Do not use timeout increases or atomics to hide an ordering defect.
    Audit sibling code/fixtures that share the failed assumption.
    Classify evidence by what it proves.

Put the detailed evidence taxonomy and examples in `docs/QUALITY.md` or an appropriate design document.

Update ExecPlan guidance if necessary so acceptance sections identify required evidence classes.

Consider PR/review guidance that asks:

    What failed?
    What invariant was violated?
    What exact path/schedule exposes it?
    What is the earliest layer that should have caught it?
    What sibling paths share the assumption?
    What evidence is deterministic?
    What evidence is merely supporting/stability evidence?

Do not require numerical repetition counts as a default quality signal.

### Milestone 8 — Preventive mechanical controls

Add mechanical checks only for stable, low-false-positive invariants.

Candidates to evaluate, not automatically implement:

- helper-specific lifecycle assertions;
- tests that verify callbacks expected by a regression were reached;
- leak detection where portable and reliable;
- repository checks for prohibited evidence wording in generated summaries only if enforceable without brittle prose policing;
- test helper APIs that make unsynchronized state difficult to access;
- race tests focused on concurrency-bearing helper packages.

Do not add a grep-based policy merely to claim automation.

If an important rule cannot be reliably encoded, document it as a review obligation instead.

### Milestone 9 — Independent review

Run an independent read-only review of:

1. audit completeness;
2. finding dispositions;
3. fixture repairs;
4. deterministic regressions;
5. evidence taxonomy;
6. Agent instruction changes;
7. mechanical controls.

The reviewer should actively search for:

- tests still able to pass without exercising the claimed path;
- synchronization that only hides race-detector output;
- lifecycle cleanup that still does not wait;
- stronger claims than the evidence supports;
- review rules that agents can satisfy cosmetically;
- checks that optimize for count or green status rather than correctness.

Address findings explicitly and retain rejected findings with rationale.

## Concrete Steps

At kickoff:

    git status --short
    git branch --show-current
    git rev-parse HEAD
    git rev-parse master

Confirm the PR #14 implementation is present in `master`; EP-OPS-001 remains active for its outstanding acceptance.

Create:

    feat/ep-qual-001

Create paired active plans:

    docs/exec-plans/active/test-architecture-evidence-governance.md
    docs/exec-plans/active/test-architecture-evidence-governance.ja.md

Run baseline:

    go run ./tools/repoctl doctor
    go run ./tools/repoctl check
    go test -race ./...

Run `repoctl plans check` and related lifecycle/provenance commands introduced by `EP-OPS-001`.

For focused test audits, prefer exact package/test commands with `-race`.

When a probabilistic repeat is useful, record it separately, for example:

    go test -race ./internal/browser/cdp -run '<test>' -count=50

and label the result:

    stability evidence only; the relevant ordering is proven by <deterministic test>

Do not use the repetition command as the primary acceptance evidence for a concurrency repair.

At final validation run at least:

    go run ./tools/repoctl check
    go test -race ./...
    go run ./tools/repoctl plans check

plus all native/integration workflows required by the touched subsystems.

Record exact workflow run IDs, native OS, toolchain/runtime versions and results.

## Validation and Acceptance

This Plan is accepted only when all of the following are directly evidenced.

### Audit acceptance

- All repository-owned concurrency-bearing test helpers are inventoried within the declared audit scope.
- Browser/CDP fixtures receive direct lifecycle and oracle review.
- Cancellation/deadline tests are searched repository-wide for cancellation-versus-completion assumptions.
- Process/controller/worker/Android/integration fixtures receive bounded lifecycle review.
- Every confirmed finding has a disposition and rationale.
- Accepted findings include sibling-exposure review.
- Rejected/deferred findings remain documented.

### Regression acceptance

- The triggering Browser/CDP test no longer has unsynchronized callback-owned state.
- Its regression deterministically exercises the cancellation/server-overlap invariant rather than depending on timeout probability.
- Server/helper cleanup provides explicit completion semantics where the test requires them.
- Sibling `mockBrowser` and related cancellation tests are reviewed.
- `eventBrowser` receives the same lifecycle assessment.
- Accepted concurrency fixes have fail-before/pass-after evidence where practical.
- Race-detector passes are recorded as supporting evidence, not substituted for deterministic schedule evidence.

### Evidence-governance acceptance

Authoritative guidance explicitly establishes:

    Evidence must be classified by what it proves.
    Quantity does not upgrade evidence class.
    Repetition is not a substitute for deterministic reproduction.

It also distinguishes at least:

- deterministic invariant evidence;
- direct native/integration evidence;
- race/static/tooling evidence;
- stability/repetition evidence;
- compilation/structural evidence.

No acceptance claim may infer native behavior from cross-builds.

No concurrency acceptance claim may infer an unforced schedule solely from repetition count.

### Agent-behavior acceptance

Agent instructions require failure analysis to proceed conceptually as:

    observed failure
    -> root cause
    -> violated invariant
    -> earliest missed prevention point
    -> sibling exposure
    -> deterministic reproducer where applicable
    -> repair
    -> regression
    -> preventive control
    -> correctly classified evidence

Instructions must discourage:

- timeout inflation as a cosmetic fix;
- atomics used solely to silence race detection when ordering remains undefined;
- CI rerun success being treated as proof that the cause is gone;
- large repeat counts being used to strengthen a claim beyond what the test forces;
- stopping after the immediately failing line without checking sibling assumptions.

### Repository acceptance

- final `repoctl check` passes;
- final full `go test -race ./...` passes;
- required integration tests pass;
- required native Windows/macOS/Linux validation passes;
- docs-check and bilingual metadata checks pass;
- independent English review passes;
- independent Japanese review passes;
- semantic parity review passes;
- independent technical review has no unresolved blocker;
- PR review is valid for final HEAD;
- Plan lifecycle/provenance gates pass;
- the Plan merges into `master` before archival.

A large number of repeated successful tests is not, by itself, an acceptance criterion.

## Idempotence and Recovery

The audit phase is read-only and may be rerun.

Do not modify code while establishing the initial finding inventory unless needed to create an isolated reproducer after the finding has been recorded.

Keep findings and dispositions durable so work can resume after interruption.

If a repair creates additional failures:

1. preserve the failure evidence;
2. determine whether it exposes another invariant or invalidates the repair;
3. update the finding ledger;
4. do not weaken the new regression solely to restore green CI.

If native CI fails on unchanged code, classify the evidence before retrying. A successful rerun does not erase the failed run.

If the branch falls behind `master`, update it using the repository's normal non-history-rewriting workflow and rerun affected baseline assumptions.

Do not force-push published history.

If `EP-OPS-001` changes lifecycle schema before this Plan starts, migrate this draft to the merged schema before promotion to active.

## Artifacts and Notes

Expected durable outputs:

- paired English/Japanese ExecPlan;
- bounded test-architecture audit inventory;
- finding/disposition ledger;
- evidence taxonomy in authoritative quality documentation;
- concise Agent workflow rules;
- deterministic regressions for accepted findings;
- repaired test helpers/fixtures where required;
- targeted repository checks where justified;
- final independent-review record;
- final retrospective.

Historical repeat counts and CI evidence remain historical. Do not rewrite completed plans merely to remove them.

The retrospective should explicitly identify any evidence patterns that previously created unjustified confidence.

## Interfaces and Dependencies

This Plan uses the EP-OPS-001 implementation merged in PR #14. It has no completion dependency on EP-OPS-001: that Plan itself needs this forward live dogfood and separate Human Validation. This avoids a circular completion requirement without declaring the bootstrap complete.

No new production runtime dependency is expected.

Test-only synchronization may use Go standard-library primitives such as channels, `sync.WaitGroup`, `sync.Once`, mutexes or atomics where semantically appropriate.

Choose synchronization based on the invariant:

- use atomic/mutex for genuinely concurrent shared state;
- use channel/barrier/WaitGroup when the test needs an explicit ordering or completion relation;
- do not confuse memory safety with lifecycle completion.

Any new repository checker belongs under the existing `tools/repoctl` architecture and must work natively on Windows, macOS and Linux without introducing a Bash, Make or PowerShell requirement.

Any durable human-facing documentation change requires corresponding English/Japanese maintenance and the repository's normal translation checks.
## Kickoff audit record (2026-09-10)

The rubric below was frozen before repairs. Classify evidence by what it proves:
forced invariant schedules; direct native/integration behavior; race/static tooling;
stability repetitions; compilation/structural checks. Repetition never upgrades
the evidence class. For each test that checks an expected rejection or error, ask whether an earlier
unrelated refusal could satisfy its assertions. Cancellation notification is not completion evidence.

Baseline at PR #14 merge `cbd84ed1d2ef4456b8a95db215461bd3d726fe59`:
`repoctl check` passed unit tests and vet but rejected the supplied Plan dependency
scalar (mapping required). Concurrent `go test -race ./...` failed on occupied
Android fixture port 5554; serial Android race passed (2.545s). Suite collision
is a hypothesis, not a demonstrated external process fault. Final PR #14 Windows
native run 34418194332/job 102687663661 failed immediate-exit inspection on a
sharing violation reading guardian completion evidence; merged does not mean
that native acceptance passed.

Bounded inventory and finding ledger, before repairs:

| ID / owner | Resources, completion, shared state and failure surface | Disposition / invariant, escape, earliest detection and prevention | Evidence |
| --- | --- | --- | --- |
| Q01 CDP fixtures | Six HTTP/WS fixture families own hijacked handlers and client reader; only mockBrowser joins handler; callback state uses atomics/mutexes | ACCEPT: join owned work before teardown; sibling lifecycle gap escaped helper review; share explicit admission/close/join ownership | Source inspection; test forcing the problematic ordering pending |
| Q02 CDP rejection/error tests | Cancellation, overflow, truncated AX and frame-mutation paths accept generic errors or omit effect checks | ACCEPT: intended failure stage must be reached; unrelated early refusal can pass; review oracle before implementation, add stage and cause checks | Source inspection; fault controls pending |
| Q03 CDP load wait | Atomic evaluation counter is sound, actual load cancellation overlap is not forced; existing 50ms cleanup observation is timing evidence | ACCEPT: force evaluation entry and cancellation, distinguish completion from notification; schedule/oracle design is earliest control | Source inspection |
| Q04 app command fixtures | SQLite/temp resources outlive success-path done receive but early Fatal can skip cancellation/release/join; cancellation, UI, browser and readiness siblings | ACCEPT: cleanup must own operation on failure paths; helper API review should establish completion before DB cleanup | Source inspection |
| Q05 execx/process fixtures | Native subprocess stop/wait/Destroy errors ignored in cleanup; detached fixtures vary in fallback containment | ACCEPT: failed termination must fail validation; cleanup oracle review, inject failed completion | Source inspection |
| Q06 descendant markers | Delayed child marker writes ignore errors; absence after sleep does not independently prove termination | ACCEPT: independent identity/completion oracle plus positive control; timing proxy escaped oracle review | Source inspection, not proof of a production leak |
| Q07 Android servers | fakeVersionServer owns listener, accept loop and unjoined per-connection handlers without deadlines; fake emulator uses fixed ports | ACCEPT lifecycle investigation; classify port collision separately until reproduced; review cleanup ownership and suite isolation | Source inspection plus failed concurrent baseline |
| Q08 Windows detached | Immediate-exit Inspect propagates transient completion-proof read errors; native helper test retries every error and hides this failure surface | ACCEPT investigation of producer/observer contract; exact sharing handle unknown; production vs oracle fault requires controlled evidence | Native CI failure and source inspection |
| Q09 control-plane/worker | Server fixture joins Run, store capacity joins callers; stream producers and early constructor failures need sibling inspection; worker helpers mostly synchronous | Continue bounded audit; no general production defect asserted; inventory separates concurrency outcomes from forced schedules | Source inspection |

Existing correctness-audit findings remain historical; this record does not
rewrite their evidence. Remaining inventory must cover all repository-owned Go
test/helper families before M2/M5 acceptance. No finding is considered repaired
solely because a suite later passes.

Additional dispositions before repairs: Q10 (ACCEPT, Flutter late-path oracle):
`TestBuildRejectsSymlinkArtifactsAndLatePathEscape` can pass on mutation failure
or before the fake build; assert successful mutation and intended refusal.
Earliest prevention is oracle/reachability review (UNREACHED_EFFECT). Q11
(ACCEPT investigation, Docker integration ownership): cleanup registration depends
on returned create JSON; missing output after allocation can lose lease cleanup.
Compare the existing Podman registry fallback and add a bounded recovery control.
No actual leaked daemon resource has been observed in this audit.

M2 inventory coverage is grouped by repository-owned helper families (source
review, not exhaustive schedule exploration): CDP HTTP/WS; app DB/operation
helpers; execx native subprocesses; Android fake TCP/emulator; control-plane
TLS/stream/DB; worker journal/CAS; repoctl Git/plan/release/docs; CLI Docker,
Podman, multi-host, process, browser, Flutter/UI; Compose runners/ports; Flutter
runner/path mutations; source Git/remotesource; instance locks; SQLite; blobstore;
assets; evidence/config/stack/domain/policy/paths; buildinfo. All are test-local
owned resources. Synchronous groups use sequential callbacks/counters and temp
cleanup; no concurrent safety is inferred for those fakes. SQLite capacity and
blobstore parallel callers drain results and join. Control-plane server cancels
and joins Run before DB close; streaming failure exits need further completion
review. Worker helpers are predominantly synchronous. CLI native fixtures own
process stop/Wait and log close; real engines require explicit opt-in. Repoctl
release fixtures own private clones/files and synchronous builds; byte-cap tests
force growth and check the mutation flag. Instance lock readiness uses stdout
barriers and process Wait; no shared mutable callback state was identified.
Source/CAS and pure boundary helpers use synchronous filesystem mutation.
Platform-specific tests supply native behavior only on their executed platform.

Q09 disposition: DEFER additional stream/constructor failure-path hardening;
closed pipes/contexts already bound these operations and no concrete false-pass
was demonstrated in this bounded review. Revisit with an injected constructor or
stream failure and an explicit producer-completion oracle before claiming a
defect. REJECT a blanket ban on sleeps or error-only assertions: those syntactic
patterns alone do not establish an invalid oracle.

### Repair evidence, 2026-09-10

- Q01/Q03: shared CDP admission/handler joins and explicit reader-exit signals.
  A no-join fault-model overlay fails `TestFixtureWorkJoinWaitsForCallback` with
  “join returned before callback completion”; `testing/synctest` establishes the
  blocked ordering. This reconstructs old behavior, not untouched-HEAD execution.
  Actual load wait forces evaluation entry, cancellation, release and completion.
- Q02: the same early unrelated error mutation passes the old truncated-gone
  oracle (0.004s) and fails all three strengthened cases (“AX truncation boundary
  not reached”). Normal focused tests pass; CDP full race passes (8.842s).
- Q04: removing operation-cleanup registration by overlay fails “resource cleanup
  preceded operation completion”. Six app callers now register cancel/join before
  assertions; early-exit cleanup is tested independently of the successful path.
- Q05: managed/native process fixture cleanup now reports failed termination or
  completion. No injected failure is claimed for every cleanup site.
- Q06: native leaf owns a TCP connection; a ready byte proves startup and EOF
  proves that fixture connection lifetime ended. Live echo/controlled-exit/native
  Wait is the positive control. Parent-only-kill overlay fails both execx normal
  and app Destroy with “descendant lifetime connection did not end”. This proves
  the tested leaf lifetime, not disappearance of arbitrary descendant trees.
  Affected full Linux race: app 47.223s; execx 6.603s.
- Q07: Android TCP fixtures close admission and accepted connections, then join
  handlers. A close-listener-only overlay fails the test with a partially sent request
  (“cleanup returned with a partial-request handler still active”). Normal focused
  tests pass; full Android race passes (2.600s). Fixed-port suite isolation remains
  a harness constraint: run these suites serially, do not relax port ownership.
- Q08: deterministic test holding a Windows DELETE handle proves the sharing-conflict
  injection before checking pending empty-Job state and release-to-absence. Bad
  proof and other read failures remain refused. Windows amd64 cross-compilation
  passes; native execution and native fail-before remain pending CI.

Mechanical-control decision: run the above tests of operation completion and
reachability in existing harness/native/race jobs. Do not add prose/sleep/count
scanners that cannot distinguish evidence classes semantically.

Remaining scope limits: inline CDP Observe fixtures join server handlers but do
not directly expose their internal client reader completion; named connection
fixtures do join readers. Full harness/race, final integration and native CI,
independent technical/EN/JA/parity review and merge remain outstanding.

### Review corrections and remaining accepted findings

Independent technical review identified two additional gaps in the new repairs:
CDP operation workers needed their own failure-path joins (reader/server joins
are insufficient), and the Docker lost-response test needed nonempty runtime
identities plus independently observed containers before cleanup. Both are
corrected. The browser mutation-fence startup wait now rejects a premature
operation result instead of waiting indefinitely; an early-return overlay fails
that assertion promptly (0.008s), and focused race passes (1.246s).

Q10: a pre-mutation build-error overlay passes the original Flutter oracle and
fails all five strengthened cases. Normal focused tests pass; Flutter focused
race passes (1.017s). Q11: fixture cleanup now enumerates its private SQLite
registry regardless of returned CLI JSON and preserves recovery state when
cleanup cannot be confirmed. No global discovery/prune authorizes destruction.
No-daemon controls cover missing/malformed response, wrong ID, incomplete state,
destroy error and corrupt registry (focused CLI race 1.282s). The real Docker
test reproducing lost CLI output verifies live containers before and absence after
cleanup; its native result is pending below. No old-code Docker fail-before is
claimed merely because the new helper API was absent.

English and Japanese policy additions received independent reader and meaning
review. The supplied Japanese Plan omitted the general-purpose model-checker
non-goal and the behind-master recovery step; both were restored. The reviewer rechecked and withdrew the M5 omission: all three fields were
already present in the Japanese schema at review time.

Q12 (ACCEPT before repair): The Android UI companion test for symlink rejection uses an empty target or invalid metadata. An unrelated missing-file/source-mismatch refusal can pass without proving symlink rejection. Invariant: start from valid companion artifacts, confirm success, then isolate directory/file symlinks and assert the intended refusal. Earliest prevention: fixture and assertion design for rejection tests; test-only UNREACHED_EFFECT, not a demonstrated product bypass.

Q12 evidence: replacing directory/file Lstat with Stat by overlay passes the
original test for symlink rejection. The strengthened valid-artifact controls reject that
same fault (directory, metadata and APK symlinks accepted). Normal uihelper tests
pass. Relative links keep file targets inside the valid companion root, avoiding
an unrelated root-escape refusal. The corpus contains 195 baseline test files in
28 package directories; three owned fixture files bring the current total to 198.
UI helper/Android UI tests use synchronous provenance/payload/dispatch fixtures;
the additional audit of fixtures for rejection tests yielded Q12. Process-runtime native
immediate-exit inspection is the direct caller that exposed Q08.

Local validation checkpoint: `repoctl doctor` and `repoctl check` passed on Linux
Go 1.27.1. `repoctl test-integration` passed with Docker/Compose. After the final
pre-cleanup container oracle, `AGENT_ENV_INTEGRATION=1 go test -tags=integration
./internal/cli -run '^TestIntegrationRegistryRecoversLostCreateResponse$'
-count=1` passed (25.594s). This is direct daemon evidence for Q11. Full uncached
race and final post-review harness are in progress; Windows/macOS native CI,
current-HEAD review gate and merge remain outstanding.

Final local race: `go test -race ./... -count=1` passed all packages (app
47.616s, CLI 41.073s, CDP 8.257s, execx 6.642s, Android 2.451s).
Post-correction independent technical review found no additional issues in CDP
worker ownership, Docker before/after resource observations or Q12 symlink
controls. Reader and semantic parity re-review passed; these are independent
agent reviews, not GitHub current-HEAD approval or maintainer acceptance.

Final post-review `repoctl check` passed format, unit tests, vet, docs, generated and architecture checks. Local implementation and validation are ready for commit and native CI. Q09 remains the explicitly deferred investigation above; no completed/archive claim is made before review gate and merge.

### Native CI correction (Q06)

Commit `d5f43cc8a3b160b6926acc0ad786c7a035d46ca8`, Verify push run
34420910331: Windows Go 1.26/1.27 failed the new descendant oracle. Process exit
reports WSAECONNRESET, not EOF, including the controlled positive exit. Also,
execx accepted its connection only after runner completion: Windows can discard
the queued connection/readiness bytes when the process dies, so accept or ready
read failed before lifetime observation. Linux/macOS success did not establish
this Windows protocol assumption. This is a test-fixture portability defect in
our Q06 repair, not evidence of a new production process leak.

Before correction: require an acknowledged startup handshake before triggering
termination/normal parent exit, and classify only EOF or the exact remote reset
as terminal connection lifetime. Timeouts, local closure and unrelated errors
must remain failures. Re-run the surviving-leaf fault control and native CI;
preserve this failed attempt. Windows process-runtime Q08 and CDP packages
passed in the same failed full run. Browser and Multi-host native runs and
Release preview passed across Windows/macOS/Linux. Linux integration passed.

PR #15 tracks this Plan. `plans provenance --plan EP-QUAL-001 --pr-body <file>`
and `plans check` passed after commit. The read-only `plans gate --plan
EP-QUAL-001 --pr 15 --repo mahcialet/agent-env` returned guarded/manual fallback:
the trusted base has no `.github/execplan-gates.json`. Do not provision policy or
merge implicitly; final acceptance/archive needs the maintainer's normal review
and merge.

Q06 decision refinement: preserve the original 300ms Command.Timeout scenario; the first repair unnecessarily expanded it to 3s. Do not introduce a dynamic Context with changing deadline semantics merely to arm a timeout after startup. Observe and acknowledge startup concurrently with Run, before parent continuation, and retain a real command timeout as direct native temporal evidence. A deadline that expires before startup is a failed test, never successful evidence of entered cancellation. Only callback/barrier ordering is classified as forced evidence.

Q06 correction implemented: asynchronous runner + accepted ready/ack precede parent continuation; only typed read resets (Windows WSAECONNRESET) or EOF count as connection termination. Deadline/local-close/refusal/write-reset/bare-reset controls remain rejected. Focused Linux race passes (app 1.261s, execx 1.416s); parent-only-kill overlays still fail on surviving leaf timeout (5.011s/5.129s). PR Verify run 34420955724 reproduced the same Windows failures on the first implementation; its Linux integration passed.

Post-Windows-correction independent technical review found no additional defect; EOF-only comments were aligned with remote-reset handling. Full uncached Linux race passed again (`go test -race ./... -count=1`: app 48.064s, execx 3.926s, CLI 42.778s). Both affected test packages cross-compile for Windows amd64; this is compilation evidence only.

Post-correction `repoctl check` passed all stages before the next native CI attempt.

Q13 (ACCEPT investigation before repair): e558891 Verify push run34421905630 integration job102698956780 failed `TestRemoteCreateNearManifestLimitRoundtrip` after62.02s with controller capacity/no online compatible host. This is a newly observed CLI fixture failure, not proof that a previous change broke manifest transport. Investigate host-registration freshness relative to expensive bundle preparation and sibling fixtures. Earliest prevention: explicit prerequisite lifetime contract and a forced stale-host control; do not inflate TTLs or hide the failed run with a rerun.

Q13 source evidence: registration precedes another near-4MiB Build plus blob upload inside flags.create; Online expires30s after last_seen and the fixture never heartbeats. The PR integration job102698969153/run34421909744 reproduced capacity failure after61.95s. The exact CI heartbeat age was not logged. Fix the liveness contract with an owned heartbeat and forced offline/refresh control, not the controller TTL. Real multi-host fixtures already run Worker.Run heartbeats; direct scheduler tests intentionally test stale hosts and must remain unchanged.

Q13 implementation evidence: the old fixture with forced stale last_seen and independently confirmed offline host fails the same capacity refusal (race24.57s). The owned heartbeat version passes the same stale-host condition plus full near-limit transport and helper tests (race30.490s). Independent review confirms shutdown order and error retention, but rejects a cancel-notified/default-select join assertion: it repeats Q01/A5 scheduling ambiguity. Replace that assertion with synctest quiescence and a cancel-only fault control before counting it as forced evidence. Q06 Windows Go1.26/1.27 native jobs in PR run34421909744 now both pass.

Q13 join oracle corrected with synctest quiescence. A cancel-only stop overlay fails “stop returned before callback completion” (0.009s); correct focused heartbeat race passes (1.014s). An initial malformed overlay had an unused-variable compile error and was not counted as evidence. Full uncached race with the heartbeat implementation passed (app50.671s, CLI43.425s); the subsequent test-only quiescence correction received the focused race above. Current corpus is199 test files after adding the owned heartbeat helper.

Q13 final independent technical/parity re-review and post-repair `repoctl check` passed. The heartbeat change is ready for native/integration CI; preserve both earlier integration failures as evidence.

### Final implementation acceptance evidence

Implementation commit `79d4d60498f7a2fdec7441a056f59c59c588f5b6`:

- [Verify PR run 34422872234](https://github.com/mahcialet/agent-env/actions/runs/34422872234) completed successfully: native Windows/macOS/Linux on Go1.26 and Go1.27, full race, Docker integration and all cross-build targets.
- [Browser native 34422872236](https://github.com/mahcialet/agent-env/actions/runs/34422872236) and [Multi-host native 34422872219](https://github.com/mahcialet/agent-env/actions/runs/34422872219) passed on all three operating systems.
- [Release preview 34422872253](https://github.com/mahcialet/agent-env/actions/runs/34422872253) passed build and native smoke on all three operating systems.
- Earlier failed runs remain recorded above. Repetition is not the basis of acceptance; each repair has the named control and its specific evidence class.

At that checkpoint only evidence/documentation reconciliation followed this SHA.
Subsequent Q14 PR-review fixes below require fresh validation.
Recheck current-HEAD CI and human review before merge. PR #15 is ready for that
review; the working Plan stays active until the normal merge/archive procedure.

Q14 (ACCEPT before repair): PR #15 automated review found four remaining test defects after successful CI. CDP canceled-call and continuous-mutation waits checked ctx.Err rather than the returned error, admitting unrelated failures once cancellation occurred (UNREACHED_CAUSE). Android partial-request and app early-exit control rescues ran only after synchronous close/t.Run returned, so a missing connection-close/cancel with join retained could deadlock before rescue (FAILURE_PATH_OWNERSHIP). These are test-only findings; strengthen returned-error oracles and give each broken-helper control an independently owned bounded release path. Record each mutation outcome; green CI was not sufficient review evidence.

Q14 controls: same unrelated returned-error substitutions after observed callback/frame mutation pass four old CDP tests (0.514s) and fail all four strengthened tests (0.515s). Six normal focused CDP tests pass under race (1.563s). App missing-cancel/retained-join mutation fails via independent watchdog (5.013s); omitted cleanup still fails ordering (0.010s). Android missing-connection-close/retained-Wait mutation now fails its explicit bounded rescue assertion (5.017s), while normal focused race passes (1.012s). Independent review and focused race verify app/Android/CDP ownership and error discrimination. An initial malformed CDP overlay failed compilation and was excluded; only corrected runtime mutations count.

Q14 final independent source/English/Japanese review found no remaining issue. Full uncached race passed (app47.446s, CLI41.503s, CDP8.138s, Android2.429s), and final `repoctl check` passed all stages. Native CI must validate this follow-up commit; earlier CI acceptance does not substitute for the current revision.


### Q14 final acceptance (2026-09-10)

Implementation commit `9988ae811e78eea61731c1f8ebce5a27e11a26b3`:

- [Verify PR 34424717606](https://github.com/mahcialet/agent-env/actions/runs/34424717606) and [Verify push 34424714380](https://github.com/mahcialet/agent-env/actions/runs/34424714380) passed, including Windows/macOS/Linux on Go 1.26/1.27, race, Docker integration and cross-builds.
- [Browser native 34424717678](https://github.com/mahcialet/agent-env/actions/runs/34424717678) and [Multi-host native 34424717628](https://github.com/mahcialet/agent-env/actions/runs/34424717628) passed on all three operating systems.
- [Release preview 34424717634](https://github.com/mahcialet/agent-env/actions/runs/34424717634) passed build and three-OS native smoke.

All four Q14 PR threads received replies with repair/control evidence and were
resolved; a fresh query found zero unresolved threads. Plan provenance passed.
These CI results supplement the forced-fault controls above; they do not prove
unforced schedules. This reconciliation changes only the bilingual Plan.
Recheck its resulting HEAD checks and current-HEAD human approval before merge.
The Plan remains active under guarded/manual fallback until merge and archive.


### Repository terminology follow-up (2026-09-11)

The user requested implementation-grounded clarification of the remaining
Japanese rejection-test shorthand across 23 documents (67 occurrences): five
guidance/design documents, nine audit documents and nine completed Plans.
This is documentation maintenance under EP-QUAL-001, on its existing branch.
The earlier local clarification of this active Plan remains part of the change.

Decision: inspect the referenced tests and implementation before naming the input,
expected refusal/error or retained state. Preserve historical revisions, commands,
results, pending work and native-evidence limits in completed Plans; editorial
clarification does not rerun or upgrade historical evidence. Update the corresponding
English wording and review parity before refreshing translation hashes.

- [x] Reconcile all 67 occurrences with available repository evidence and review the 23 bilingual document pairs; retain the A15 limitation below (2026-09-11).
- [x] Run documentation/Plan checks and record the inspected evidence and limitations (2026-09-11).

The following ledger covers all 23 documents; AGENTS/QUALITY and ARCHITECTURE/ADR share rows.
Audit slugs refer to `docs/audits/repository-correctness/`; completed slugs refer to
`docs/exec-plans/completed/`. Both language files were edited.

| Document | Inspected repository evidence | Clarified meaning |
| --- | --- | --- |
| AGENTS / QUALITY | `internal/browser/cdp/fixture_lifecycle_test.go` | Reach the tested condition and discriminate the returned cause. |
| ARCHITECTURE / ADR 0005 | `tools/repoctl/main.go`; `tools/repoctl/main_test.go` | Deliberately forbidden imports, including nested Flutter packages. |
| design-docs/browser-cdp-automation | `internal/config/browser.go`; `internal/config/browser_test.go` | Reject missing bindings and invalid protected switches. |
| audits/current-compose-release | `tools/repoctl/release.go`; `tools/repoctl/release_path_test.go` | Detect real checkout paths without flagging module paths. |
| audits/current-process-browser | `internal/browser/cdp/client_test.go`; `internal/browser/cdp/actions.go`; `internal/cli/browser_native_test.go` | Reject identity mismatch and targets obscured by overlay elements. |
| audits/supplemental-cli | `internal/cli/browser_native_test.go` | Check required output and absence of misleading output after successful mutations. |
| audits/current-control-plane | `internal/app/readiness.go`; `internal/app/readiness_safety_test.go` | Unconfirmed termination/output stops retries and preserves cleanup barriers. |
| audits/current-mobile | `internal/runtime/android/adb.go`; `internal/runtime/android/adb_test.go` | Do not send operational ADB commands after incompatible/malformed server responses. |
| audits/history-mobile | `internal/runtime/android/adb_test.go`; `internal/app/application_identity_test.go` | Requirement-specific identity/path/environment failures and no-effect assertions. |
| audits/history-process-browser | `internal/app/plan_process_test.go`; `internal/app/browser_review_test.go`; `internal/execx/managed_windows_test.go` | Secret literals, changed manifests and native identity checks; retain helper-only limits. |
| audits/documentation | `tools/repoctl/translation_review_test.go`; `tools/repoctl/fragment_audit_test.go`; `tools/repoctl/main.go` | Missing/hidden links, invalid translation exceptions and fake heading targets. |
| audits/matrix | `internal/browser/cdp/supplemental_audit_test.go`; `internal/browser/cdp/snapshot_test.go`; `internal/browser/cdp/snapshot.go` | Changed checked/pressed state, navigation and the preceding subsystem checks. |
| completed/persistent-process-runtime | `internal/execx/detached.go`; `internal/execx/managed_test.go`; `internal/execx/managed_windows_test.go`; `tools/repoctl/main_test.go` | Observation/termination with invalid identity, PID reuse and forbidden imports. |
| completed/android-ui-observer | `internal/app/ui.go`; `internal/app/ui_test.go` | UI refusal/failure behavior and out-of-range coordinates. |
| completed/browser-cdp-automation | `internal/config/browser_test.go`; `internal/browser/cdp/client_test.go`; `tools/repoctl/main_test.go` | Profile violations, controlled transport mismatches and forbidden dependencies. |
| completed/flutter-android-runtime | `internal/config/application_test.go`; `internal/app/applications_test.go`; `internal/app/application_identity_test.go`; `internal/runtime/android/application_test.go`; `tools/repoctl/main_test.go` | Invalid manifests, unconfirmed termination, evidence/launch failures and forbidden dependencies. |
| completed/standalone-release-finalization | `tools/repoctl/release_source.go`; `tools/repoctl/release_source_test.go`; `tools/repoctl/release_e2e_test.go`; `tools/repoctl/release_workflow_test.go` | Invalid release inputs, modified archives and publication-gate bypass mutations. |
| completed/standalone-distribution | `tools/repoctl/release_source_test.go` | Git mismatch/dirty input rejection and private checkout isolation. |
| completed/multi-host-control-plane | `internal/controlplane/server/server_test.go`; `internal/controlplane/store/store_test.go`; `internal/blobstore/store_test.go`; `internal/remotesource/package_test.go`; `internal/cli/multihost_integration_test.go` | ACL, cleanup proof, CAS/source validation and helper environment-error controls. |
| completed/repository-correctness-audit | `internal/browser/cdp/supplemental_audit_test.go`; `internal/browser/cdp/snapshot_test.go`; `tools/repoctl/fragment_audit_test.go` | Defect reproduction, incomplete Browser observations and fragment checks; Android A15 limit below. |
| completed/repository-correctness-review | `internal/browser/cdp/page_create_review_test.go`; `tools/repoctl/fragment_audit_test.go` | Missing target type cannot prove absence; valid/invalid heading anchors. |


Review corrected three misleading candidate phrasings: successful CLI mutations
must not become failed-operation scenarios; completed Windows Jobs can correctly
report absence despite historical PID reuse; AGENTS now requires reaching the
tested condition, not performing a forbidden effect. Independent changed-passage
review covered the five guidance/design pairs and the prior active-Plan edits;
the integrating editor reviewed the audit/completed-Plan changes.

Limitation: the dedicated Android partial/truncated-wait fixture behind historical
A15 was not directly identified. Its original historical description remains,
separate from the directly inspected Browser absence tests. Past CI durations and
OS acceptance were preserved, not rerun or independently recertified.


Validation for this documentation change: `repoctl docs-check`, `repoctl plans check` (three Plans valid) and `git diff --check` passed. The intermediate
documentation/Plan checks rejected translation hashes while those pairs
were still being edited; after parity review and hash updates they passed.
All 48 changed files are Markdown. Comparing inline code, link targets and
headings against HEAD found no differences outside the intentionally extended
active Plan. The repository Markdown search found no remaining Japanese shorthand
occurrences from the 67-entry inventory.

Focused current-source tests passed on Linux Go 1.27.1:

```text
go test ./tools/repoctl ./internal/config ./internal/browser/cdp -run '^(TestArchitectureBoundaries|TestBrowserManifestContract|TestBrowserManifestNegativeFixtures|TestBrowserRequiresProcessRuntime|TestBrowserAbsentPreservesLegacyCanonicalShape|TestFixtureCancellationErrorDiscriminatesReturnedCause)$' -count=1
```

Results: repoctl 0.021s, config 0.016s, CDP 0.003s. This supports the inspected
guidance examples; it is not a new full-suite or native acceptance claim.


### Dependency terminology follow-up (2026-09-11)

The next requested terminology pass covers 18 occurrences in 14 Japanese
documents and their English counterparts. Preserve the earlier uncommitted
editorial changes. This remains documentation maintenance under EP-QUAL-001.

Decision: distinguish component selection from Compose service selection and
resource retention. `internal/stack/resolve.go` includes roots plus every direct
and indirect `DependsOn` dependency in deterministic order. `policy.Services`
similarly follows Compose `depends_on`; `pruneConfig` then keeps resources
referenced by the resulting services. Do not describe resource retention as
starting arbitrary additional services, or omit the roots from the selected set.

- [x] Complete the 14 bilingual document revisions and review their evidence/parity (2026-09-11).
- [x] Run focused resolver, policy, Compose and application tests (2026-09-11).
- [x] Complete documentation/Plan checks and preserve the validation scope (2026-09-11).

Focused validation on Linux Go 1.27.1:

```text
go test ./internal/stack ./internal/policy ./internal/runtime/compose ./internal/app -run '^(TestClosure|TestDeterminismAndNoMutation|TestSelectedClosureAndHazards|TestRenderSelectedClosureAndPolicy|TestPruneRemovesUnselectedCleanupTargets|TestMobilePlanStackClosure|TestPlanClosureAndNoAllocation)$' -count=1
```

All four packages passed (stack 0.005s, policy 0.002s, Compose 0.002s, app 0.010s).
This checks current selection/pruning examples; it does not repeat historical
native Podman or Flutter acceptance.


Coverage: ARCHITECTURE, QUALITY, roadmap, ADR 0003; design documents
compose-runtime, compose-providers, core-beliefs, lease-control-plane; product
specifications agent-env-mvp, cli-contract, compose-providers, manifest-v1;
the current-control-plane audit and completed flutter-android-runtime Plan.
All 18 occurrences were inspected; code identifiers and heading anchors remain unchanged.

Evidence: `internal/stack/resolve.go` and `resolve_test.go`;
`internal/app/plan.go`, `plan_test.go` and `applications_test.go`;
`internal/policy/policy.go` and `policy_test.go`;
`internal/runtime/compose/compose.go`, `compose_test.go`, `prune.go`,
`prune_test.go` and Podman's shared validation call in `podman.go`.
Independent changed-passage review of ARCHITECTURE, QUALITY, roadmap and the
Flutter Plan found one subject drift: QUALITY must retain the observed lease-ready
result, rather than claim an individual service-ready result. Both languages now
retain that subject. The integrating editor reviewed the remaining ten pairs.

Final dependency-wording checks: `repoctl docs-check`, `repoctl plans check`
(three Plans valid) and `git diff --check` passed. The repository Markdown
search found no remaining occurrences of the targeted Japanese term.


QUALITY volume wording follow-up (2026-09-11): clarify the recorded test's
Dockerfile `VOLUME` declaration, Podman-created anonymous volumes, observed
`Anonymous: true`/empty `Labels`, and absence after destroy.
Source inspection: `internal/cli/podman_integration_test.go` (image creation,
attached-volume inspection and post-cleanup listing), and
`internal/runtime/compose/podman_anonymous.go`. This is an editorial
clarification of the historical run, not a new native execution.


### Acceptance intent tables (2026-09-11)

The user requested that dense acceptance lists explain what each condition aims
to establish. Restructure QUALITY's publication gates, persistent-process lifecycle
acceptance and multi-host native-fixture scope into item/check/reason tables.
Keep prerequisites, historical results and native/physical-host limits outside
the tables, and preserve the earlier editorial changes. This does not change
product behavior or introduce a repository-wide table-format rule.

Rationales come from existing checks, not inferred implementation history:
`.github/workflows/release.yml`, `tools/repoctl/release_workflow_test.go`,
`release.go` and `release_smoke.go`; `internal/cli/process_native_test.go`,
`process_compose_integration_test.go`; `internal/app/process_lifecycle_test.go`;
`internal/runtime/process/process_test.go`; and
`internal/cli/multihost_integration_test.go`.
The process fixture proves two simultaneously live leases, not simultaneous create
calls. Persistence-failure cleanup remains conditional on ownership evidence.
The browser-shaped helper does not establish Browser/CDP feature correctness.
The multi-host fixture remains two worker roots on one host.

- [x] Complete independent English/Japanese reader and semantic-parity reviews (2026-09-11).
- [x] Run documentation checks and record focused validation with its limits (2026-09-11).


Independent reviews retained all original conditions. Revisions clarified
whole-lease placement as all runtimes on one worker, renewal as preservation of
the managing controller/placement rather than the user-facing Owner label, and
smoke-test scope as CLI startup/dependencies rather than all provider runtimes.
English/Japanese observation wording now explicitly includes state and logs.
Historical successful runs remain separate from the tables.

Focused current-source checks passed on Linux Go 1.27.1:
`TestReleasePublicationGate` (repoctl 0.004s);
`TestProcessIdentitySaveFailureAndPartialStartCompensate`,
`TestProcessUnknownOwnershipQuarantinesAndRecovers` and
`TestProcessReadinessUsesRecordedNumericEndpoint` (app 0.036s);
`TestReservedPortOccupationPreventsLaunch` and
`TestReceiptFailurePreservesReturnedIdentity` (process adapter 0.017s).
All were selected with `go test`, an anchored name pattern and `-count=1`.
The initial combined selection matched no process-adapter tests; that result was
not counted as validation, and the two correct adapter test names were run
separately. No new native multi-host, Compose or release-candidate run is claimed.

Table restructuring validation: `repoctl docs-check`, `repoctl plans check`
(three Plans valid) and `git diff --check` passed.


### Process test-scope tables (2026-09-11)

The user requested the same intent-first structure for QUALITY's preceding
config/adapter/native-primitive paragraph. Replaced it with three tables
(5 configuration, 7 adapter and 3 OS-process rows), preserving the existing
requirement to check that existing Android detached-process behavior remains correct and the separate lifecycle table.
Source-relative path confinement is distinguished from host PATH lookup.
Prelaunch redaction evidence, post-launch secret changes, and root-exit/descendant
survival are separate checks with separate reasons.

Evidence: `internal/config/process_test.go`;
`internal/runtime/process/process.go` and `process_test.go`;
`internal/execx/managed_test.go` and `detached_test.go`.
Focused uncached config and adapter tests passed (0.008s/0.136s), covering
manifest variants, YAML presence, legacy serialization, expansion, path/identity
checks, bounded logs and durable redaction. Native process tests were inspected,
not rerun; this editorial change does not renew prior OS acceptance.

- [x] Finish independent English/Japanese/parity review and final documentation checks (2026-09-11).


Independent English, Japanese and parity passes found two overbroad phrasings:
ports/readiness remain optional, so checks apply when configured; portable-name
checks specifically cover process runtime names used as directories. Both were
corrected against `internal/config/process.go`. Final `repoctl docs-check`
and `git diff --check` passed.


### First-reading clarity across documentation (2026-09-11)

The user requested clearer explanations of regression terminology and dense prose.
Preserve all earlier uncommitted editorial changes. This pass reviews QUALITY in
full and the passages using the Japanese shorthand in repository documentation;
it does not claim a complete readability rewrite of every repository document.
For those passages, distinguish tests preserving existing behavior, tests detecting
recurrence of a repaired defect, and historical mappings to current checks.
Preserve test names, links, historical results and unverified-environment limits.

Decision: explain the check and its purpose in ordinary language. Use tables for
parallel acceptance conditions and connected prose for mechanisms and limitations.
Keep existing headings where changing them would break anchors; explain any
retained term directly below its heading. No product behavior or test requirement
is changed by this editorial work.

Discovery: the multi-host design's Japanese expiry description implied completed
cleanup. `TestControllerServeQueuesExpiryWithoutPolling` instead verifies a queued
destroy request while the worker is offline and the lease remains READY. The test
polls SQLite to observe this, but does not use a worker poll or HTTP request to
trigger expiry. Both languages now distinguish that queueing evidence from
completed cleanup. This clarification does not rerun historical acceptance.

- [x] Complete the QUALITY full-document pass and the terminology passage inventory.
- [x] Complete independent reader/parity review and documentation validation.


Terminology inventory: 43 Japanese documents and their English counterparts:
QUALITY; PORTABILITY; design documents compose-providers, multi-host-control-plane
and persistent-process-runtime; both active Plans; 13 repository-correctness audit
documents; and 23 completed Plans. Each targeted passage is read in context.
QUALITY receives a full section-by-section readability pass followed by a global
pass; the other documents receive a targeted passage review, not a full-prose rewrite.
The audit subset contained 188 occurrences on 184 lines. Historical mappings now
say “current verification” where they include documentation, CI or unresolved
coverage as well as tests, so the label does not imply a reproduced old defect.

Focused current-source checks passed on Linux Go 1.27.1:
`TestControllerServeQueuesExpiryWithoutPolling`, `TestPodmanBridgeNativeRoundTrip`,
`TestCompletedStackedDependencySurvivesBranchDeletion`,
`TestStackedProvenanceValidatesSeparateHistories`, and
`TestHumanContractAcceptsCleanCRLFCheckout`, selected with an anchored test-name
pattern and `-count=1`. Results: repoctl 0.501s, CLI 1.016s, Compose 0.264s.
This validates the inspected examples; no new full-suite, real-engine or
Windows/macOS acceptance is claimed.

The completed-Plan subset contained 149 occurrences across 23 pairs: 148 were
rewritten and one protected heading was retained with an explanation immediately
below it. In the bilingual-review history, the term described a repair breaking
existing behavior, not a test. In the real-writer history, source inspection
identified injected lock loss rather than lock acquisition. Preserve both meanings.
Comparisons against HEAD found unchanged inline literals, headings and link targets
in the affected audit and completed-Plan documents; completed-Plan numeric sequences
also remained unchanged, excluding translation hashes.


Independent reviews: the root-owned passages received English-reader,
Japanese-reader and semantic-parity passes with no findings. The completed-Plan
review found repeated Japanese phrases, a nonparallel English list, and an A22
summary that could imply all 186 historical rows had defect-detection tests. These
were corrected; the summary now includes implementation, verification evidence
and its limits. QUALITY's independent full English/Japanese/parity review caught
three scope losses: non-Windows interop had become Linux-only; identifying the
earliest irrelevant rejection path had become merely checking for any such path;
and lifecycle compensation had become creation-only. All three were corrected,
including the compensation overview in both languages. These findings show why
terminology simplification needs meaning review as well as hash validation.


The independent audit-passage review found no semantic issue and one awkward
Japanese connective, now corrected. Final documentation validation passed:
`docs-check`, `plans check` (3 valid Plans), and `git diff --check`.
This editorial work remains uncommitted and unpushed; it does not complete the
Plan’s implementation/review/merge lifecycle or replace historical acceptance.


### Japanese register consistency (2026-09-11)

The user approved the preceding register review's recommendations. The review
scanned 85 Japanese documents and checked 18 candidates in context. Normalize
unexplained switches within prose to each document's predominant register.
Preserve compact table/checklist forms, quoted text, technical literals, tense,
modality and historical evidence. Retain the guide/history distinction in the
audit index and the planning/execution-record distinction in the Flutter Plan;
normalize inconsistencies within those regions rather than flattening them.
Review corresponding English meaning; Japanese politeness changes do not require
inventing new English content. This is a local editorial decision, not a new
repository-wide writing rule. Existing uncommitted changes remain in place.

- [x] Apply the approved register corrections and compare protected content.
- [x] Complete independent review and documentation checks.


Completed: 17 Japanese documents corrected; the audit index required no change.
Independent reviewers checked the narrow before/after diffs, corresponding
English meaning, tense and requirement strength. No unresolved findings remain.
The Flutter Plan's compact numbered procedures were retained in their original
form during integration; its historical-summary sentence was normalized to the
surrounding plain register. Protected literals, headings, links and numerical
evidence were unchanged. `docs-check`, `plans check` (3 valid Plans), and
`git diff --check` passed. No product code changed; no new native acceptance was
claimed. Commit and push were not performed.


### Post-destroy cache wording correction (2026-09-11)

Implementation consistency review found one confirmed editorial error in the
completed multi-host round-two Plan: a disproved cache-removal concern had been
rephrased as a repaired defect's recurrence test. With user authorization, both
languages now describe a retained test of post-destroy cache reuse with an empty
CAS. `internal/worker/postdestroy_cache_review_test.go` verifies subsequent
logs/artifact/reconcile/repeated destroy and a single process start. Keep the
separate source-diff/reuse fail-before evidence unchanged. No product behavior or
historical validation result is changed.


### Editorial commit authorization (2026-09-11)

The user authorized committing and pushing the accumulated documentation work.
The delivery includes terminology explanations, acceptance rationale tables,
Japanese register consistency, and the implementation-review correction above.
Earlier notes that edits were uncommitted describe those earlier checkpoints.
Keep this Plan active pending its existing review/merge requirements; publishing
these documentation edits does not supply new native acceptance evidence.
