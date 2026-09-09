---
status: completed
owner: maintainers
last_verified: 2026-09-09
---

# Audit repository-wide correctness invariants after Browser/CDP merge

[日本語](repository-correctness-audit.ja.md)

This ExecPlan is a living document. Maintain it according to `docs/PLANS.md`.

Expected branch: `audit/repository-correctness`.

PR #10 (`feat: add lease-owned browser CDP automation and snapshots`) is a hard
prerequisite. Do not begin until PR #10 is merged into `master`.

At audit start:
1. fast-forward `master`;
2. record the exact post-PR-#10 merge revision below;
3. create `audit/repository-correctness`;
4. run baseline harness/race/native checks;
5. freeze the audit target revision for the review-only phase.

Audit target revision: `031869c8b9073b8e23bc17fbc55243666a52f557`

## Purpose / Big Picture

Perform a repository-wide adversarial correctness audit of all implemented
agent-env behavior after Browser/CDP merges. This is not a feature plan.

The audit targets defects that survive feature-local reviews, especially:
boundary/off-by-one semantics; false completeness; false ownership/absence;
state-transition errors; persistence after external effects; cancellation and
lock-loss windows; cleanup proof; stale observations; concurrency/resource
reuse; redaction/truncation; path safety; and native OS differences.

Canonical boundary example:

    limit = 2048 AX nodes
    2047 -> complete
    2048 -> complete
    2049 -> truncated

Invariant:

`truncated == true` means at least one valid item/frame was omitted. Returning
exactly `limit` items does not itself prove truncation.

## Scope

Review all production behavior on the frozen post-PR-#10 master, including:
config/domain/app/store; Git source/worktrees; Docker Compose; Podman if merged;
Android Emulator; Flutter Android; Android UI observer; persistent process;
Browser/CDP; execx; evidence/redaction; paths; readiness/endpoints; reconcile;
destroy/GC/quarantine; standalone release/buildinfo/assets; and mechanical claims
in durable documentation.

Out of scope: new iOS/browser/Podman/multi-host features, aesthetic refactors,
performance-only work, and architecture replacement without a correctness defect.

## Audit Method

### Phase A — Review only

No product-code modifications. Record findings against the frozen revision.
Only this plan and explicit audit reports may change.

### Phase B — Disposition

Every finding receives exactly one:
- ACCEPT
- REJECT
- DEFER
- DUPLICATE

REJECT requires evidence. DEFER requires severity, risk, rationale and follow-up.
DUPLICATE links the canonical root-cause finding.

### Phase C — Remediation

For every ACCEPT finding:

    reproducer/failing fixture
      -> regression test
      -> minimal correctness fix
      -> focused verification
      -> full verification
      -> resolved evidence

Do not opportunistically fix undispositioned findings.

Completion does not require zero findings. It requires complete audit coverage,
all findings dispositioned, all ACCEPT findings resolved, no unresolved
Critical/High findings, explicit follow-ups for deferred findings, and final
verification. In addition, 100% of ACCEPT findings must have an escape analysis
and earliest-preventable-stage classification; every recurring defect class must
have a preventive guardrail or an explicit documented reason why one is not
practical in this plan.

## Finding Format

Stable IDs:

    AUDIT-<AREA>-NNN

Areas include BOUNDARY, STATE, OWNERSHIP, PERSIST, CANCEL, CLEANUP, CONCURRENCY,
STALE, REDACTION, PATH, PORTABILITY, RELEASE, SECURITY and DOCS.

Each finding records:
ID; Severity; Disposition; Invariant; Location; Trigger; Observed behavior;
Expected behavior; Impact; Existing coverage; Reproducer; Regression; Resolution;
Verification; Related findings.

Separate observed fact, root-cause hypothesis and proposed repair.

## Severity

Critical:
- unrelated/sibling resource deletion/termination;
- wrong engine/device/process/browser/page action;
- clear secret exposure through a path claimed redacted;
- ambiguous identity authorizing destructive action.

High:
- incomplete cleanup marked RELEASED;
- ambiguous identity treated owned/absent;
- successful effect lost after persistence failure;
- stale action can hit another target;
- source/state deleted while a possibly-live process still uses it;
- provider/connection changes redirect cleanup.

Medium:
- complete exact-limit evidence marked truncated;
- valid action incorrectly refused;
- non-destructive READY/DEGRADED misclassification;
- retry/timeout/off-by-one defects.

Low:
- harmless metadata/diagnostic inaccuracies without correctness impact.

## Core Invariants to Audit


### Historical review corpus

Completed review/follow-up ExecPlans are audit inputs, not dead history.

Inventory at minimum:

- files named `*review*.md` under `docs/exec-plans/completed/`;
- completed feature plans containing review/finding/thread follow-up sections;
- PR review thread IDs/links referenced by those plans;
- `Surprises & Discoveries`, failed approaches, independent-review notes and
  retrospective sections that explain why the original test suite missed a
  defect.

For every historical review finding, extract:

```text
Historical finding ID/source:
Original invariant:
Original defect:
Original severity/impact, if known:
Earliest preventable stage:
Detection opportunities that existed:
Why it escaped each earlier stage:
Why existing tests/review missed it:
Missing guardrail:
Preventive control that should catch this class earlier next time:
Regression added at the time:
Current regression location:
Current production location:
Still applicable? yes/no
Current test still proves the invariant? yes/no
Current test exercises the real public/full entry point? yes/no
Same-pattern recurrence locations:
Current audit finding IDs, if any:
Guardrail status: existing / added by this audit / deferred
```

Classify `Earliest preventable stage` using this common pipeline:

```text
S0  Product specification / acceptance criteria
S1  Design / invariant definition
S2  Implementation-local unit test
S3  Package/component integration test
S4  Cross-component integration test
S5  Native / real-provider integration
S6  Repository harness / static validator
S7  PR author self-review
S8  Independent/adversarial PR review
S9  Post-merge repository audit
```

The stage is the earliest realistic point where the repository already had enough
information to prevent or detect the defect. Do not select an earlier stage merely
because hindsight makes the defect obvious. Record concrete detection opportunities
that actually existed at that time.

The goal is not to re-litigate already-fixed code. The goal is to answer:

1. Does the historical regression still exist and still fail if the defect is
   reintroduced?
2. Has refactoring moved the behavior so the old regression no longer exercises
   the real entry point?
3. Has the same defect pattern reappeared in another subsystem?
4. Was the original fix local while the underlying invariant should have been
   repository-wide?
5. Did the historical review reveal a weakness in test methodology that still
   exists elsewhere?
6. What was the earliest realistic stage that could have stopped the defect?
7. Which concrete detection opportunities existed before merge, and why did each
   fail to detect it?
8. Was the failure caused by missing specification, missing invariant, weak oracle,
   fixture coupling, helper-only coverage, missing composition/native coverage,
   missing failure injection, or review-process weakness?
9. What preventive guardrail would make the same defect class fail earlier next
   time, ideally before S8 PR review?

Historical review replay therefore produces two outputs: correctness evidence for
the present code and defect-escape evidence for improving the development process.

Known historical patterns include:

- positive fixtures and permissive tests sharing the implementation's wrong
  assumptions instead of independently mutating each requirement;
- tests stopping at a helper/Doctor path while the following full entry point
  had different dependencies or behavior;
- global inventory/filtering accidentally suppressing unrelated resource kinds;
- runtime readiness sharing the wrong timeout/budget;
- validation occurring after an earlier side effect such as store creation or
  reservation;
- cleanup stopping after one runtime failure instead of continuing independent
  safe cleanup while preserving global barriers;
- native process observation races at OS boundaries;
- protocol-specific logic (for example TCP reachability) incorrectly applied to
  another protocol (for example UDP);
- optional frontend/tooling accidentally becoming required by a lower-level
  inventory/inspection path.

Historical review replay is distinct from current-code review: a historical
finding can be correctly fixed at its original location yet still generate a new
audit finding when the same root pattern recurs elsewhere.

### Defect escape analysis

Every current `ACCEPT` finding and every materially relevant historical finding
must receive an escape analysis. The central question is not only "why was the
code wrong?" but:

> Why did this defect survive until the stage where it was finally discovered,
> when an earlier repository stage should have been able to reject it?

For each accepted/current finding record:

```text
Detected at stage:
Earliest preventable stage:
Detection opportunities before discovery:
Escape reason(s):
Missing/weak test oracle:
Missing composition/native/failure-injection coverage:
Review-process gap, if any:
Preventive guardrail:
Guardrail implementation/evidence:
Expected future detection stage after guardrail:
```

Use escape-reason categories where they help aggregation:

- `SPEC_GAP` — required behavior was not explicit enough to derive a test;
- `INVARIANT_GAP` — design did not state the safety/completeness invariant;
- `BOUNDARY_GAP` — tests omitted exact/min/max boundary cases;
- `ORACLE_COUPLING` — test expectations/fixtures shared the implementation's wrong assumption;
- `HELPER_ONLY` — helper passed but the public/full entry point was not exercised;
- `COMPOSITION_GAP` — components were correct alone but interaction was untested;
- `FAILURE_INJECTION_GAP` — effect/persistence/cancellation failure window was untested;
- `NATIVE_EVIDENCE_GAP` — fake/cross-build evidence substituted for real native behavior;
- `CONCURRENCY_GAP` — independent-process or race behavior was not exercised;
- `NEGATIVE_FIXTURE_GAP` — only positive/golden fixtures existed;
- `REVIEW_CHECKLIST_GAP` — known invariant had no explicit self/adversarial-review prompt;
- `HARNESS_GAP` — repository tooling could have mechanically enforced the rule but did not.

A finding may have multiple escape reasons. Prefer the earliest actionable cause
over blaming the final reviewer.

Preventive controls may be:

- product/design invariant;
- requirement-specific negative fixture;
- boundary helper (`limit-1`, `limit`, `limit+1`);
- full-entry-point integration fixture;
- failure-injection helper;
- native real-provider CI;
- repoctl/static/architecture check;
- reusable ownership/absence-proof helper;
- PR author self-review checklist;
- adversarial review checklist.

For repeated defect classes, a local regression alone is insufficient unless the
audit explains why no broader guardrail is practical.

### Boundary/exact-limit semantics
For every meaningful numeric cap test 0, 1, limit-1, limit, limit+1.
Audit Browser AX/DOM, Android UI, console/network/log/artifact caps, redaction
buffers, retries, timeouts, ports, page/frame counts, SQLite retries and release
limits. `truncated` requires actual omitted evidence.

### Positive proof
Audit READY, owned, healthy, complete, clean, absent, released, safe-to-delete,
safe-to-retry and same-target decisions. These must be positively proven where
they authorize later effects, not merely “no failure observed”.

### State transitions
Enumerate required/forbidden evidence and persistence barriers for:
ALLOCATING->READY, READY->DEGRADED, *->QUARANTINED,
active->releasing->released and failed-create compensation.

### Persistence around external effects
For each effect inspect:

    persist intent -> effect -> persist exact identity/result -> later effects

Fault-inject/store-fail around worktree create/remove, Compose/Podman up/down,
Emulator start/stop, Flutter build/install/launch, adb reverse, Android UI input,
persistent process start/terminate, Browser/CDP mutation, and release publication.

### Ownership/identity
Audit PID/process-group/Windows Job reuse, port reuse, project/resource names,
Podman connection changes, Emulator serial/AVD identity, CDP port reuse,
page/target/node replacement, and symlink/path replacement. Names alone are not
ownership proof.

### Cleanup proof
`cleanup command succeeded != resource absence proven`.
Check Docker/Podman down, process termination, Emulator kill, browser disconnect
and file removal. Uncertain cleanup must retain evidence/state/reservations and
quarantine rather than delete dependencies.

### Cancellation/deadline/fence loss
Check before/during/after-effect cancellation, persistence after effect,
cleanup cancellation and operation-lock loss. Ensure no “one more effect after
lock loss”. Timeout never proves the external effect did not happen.

### Concurrency/reservations
Use independent processes/connections where cross-process correctness is claimed.
Review same lease, sibling leases, same source, cold SQLite init, ports, emulator
slots, process ports, assets, release output and cleanup-vs-action races.

### Stale semantics
Android UI and Browser actions must revalidate exact current target and uniquely
re-resolve nodes before input. No coordinate fallback. `gone`/absence waits must
not treat truncated evidence as complete.

### Redaction/bounded evidence
Cover secrets across chunk boundaries, truncation boundaries and multibyte text;
redaction expansion versus persisted caps; fail-closed behavior when redaction
proof is required. Screenshots/raw mutable state are not pixel/text-redactable by
the same mechanism.

### Paths/filesystems
Audit absolute/relative paths, `..`, symlinks, case folding, Windows drive/UNC,
spaces, non-ASCII, state/source boundaries, archive traversal and runtime/profile
directories. Review TOCTOU where destructive path checks matter.

### Cross-platform semantics
Verify Windows/macOS/Linux separately for process identity/tree cleanup, Windows
Job/guardian completion, macOS observation, open-handle deletion, case/path
behavior, executable extensions and graceful termination. Cross-build is compile
evidence only.

### Release correctness
Audit tag/version/source identity, clean-tree policy, deterministic archives,
checksums/manifests, validated-byte publication, archive path safety and no
rebuild after validation.

### Docs claims versus enforcement
Search durable docs for must/never/always/enforced/rejects/guarantees. Mechanical
claims require implementation plus positive and negative tests, or the prose must
be corrected.

## Progress

- [x] (2026-09-09) PR #10 merged; fast-forwarded master (already current), created audit/repository-correctness and froze Phase A at 031869c8b9073b8e23bc17fbc55243666a52f557. User-supplied untracked bilingual audit plans are preserved and adopted. No product changes are allowed during Phase A.


- [x] (2026-09-09) Merge PR #10 and record the exact audit target revision.
- [x] (2026-09-09) Create `audit/repository-correctness`.
- [x] (2026-09-09) Run baseline repoctl/docs/race/native/integration checks.
- [x] (2026-09-09) Freeze Phase A target revision.
- [x] (2026-09-09) Define audit matrix and finding-report path/ID convention.
- [x] (2026-09-09) Inventory completed review/follow-up ExecPlans and referenced PR findings.
- [x] (2026-09-09) Build a historical review corpus mapping original findings to current
      regressions, production paths and same-pattern recurrence searches.
- [x] (2026-09-09) For every material historical finding, record earliest preventable stage,
      detection opportunities, escape reason and preventive guardrail.
- [x] (2026-09-09) Replay/inspect historical regressions and identify stale or weakened
      coverage before product modifications.
- [x] (2026-09-09) Aggregate historical escape reasons by S0-S9 stage and category.
- [x] (2026-09-09) Review boundary/exact-limit semantics.
- [x] (2026-09-09) Review state/positive-proof semantics.
- [x] (2026-09-09) Review persistence/effect ordering.
- [x] (2026-09-09) Review ownership/identity proof.
- [x] (2026-09-09) Review cleanup/absence proof.
- [x] (2026-09-09) Review cancellation/deadline/lock-loss behavior.
- [x] (2026-09-09) Review concurrency/reservations.
- [x] (2026-09-09) Review stale observation/action behavior.
- [x] (2026-09-09) Review redaction/truncation interaction.
- [x] (2026-09-09) Review path/filesystem safety.
- [x] (2026-09-09) Review cross-platform semantics.
- [x] (2026-09-09) Review release correctness.
- [x] (2026-09-09) Review docs enforcement claims.
- [x] (2026-09-09) Complete Phase A without product-code modifications.
- [x] (2026-09-09) Disposition every finding.
- [x] (2026-09-09) Confirm no Critical/High finding is silently deferred.
- [x] (2026-09-09) Add regression evidence before fixes where mechanically possible.
- [x] (2026-09-09) Resolve all accepted Critical/High findings.
- [x] (2026-09-09) Resolve accepted Medium/Low findings in scope.
- [x] (2026-09-09) Promote recurring patterns into shared test/check/helpers where justified.
- [x] (2026-09-09) Perform independent final read-only re-review.
- [x] (2026-09-09) Run final full harness/race/native/integration matrix.
- [x] (2026-09-09) Verify unresolved Critical=0 and High=0.
- [x] (2026-09-09) Complete bilingual Outcomes & Retrospective.
- [x] (2026-09-09) Move both plans to `docs/exec-plans/completed/`.

## Surprises & Discoveries

- 2026-09-09 completion checkpoint: all earlier pending checks below have been reconciled. Final19 repairs, independent review, full harness/race, native CI34295144985/34295144958 and final release verification pass. Archived both language plans after filling all38 acceptance rows. Earlier entries retain their observations at that stage, not current open work.

- 2026-09-09 final reconciliation: candidate6872286 Verify34294068659 and Browser native34294068663 pass completely, as do real local integrations and release verification. Final historical-source refresh exposed four postmerge PR10 comments3963154175/4182/4186/4191 absent from its completed plan. Supplemental review-only reproductions now cover page-create128 limit, mutation table output, ignored AX wait and pressed-state fingerprint. Do not archive until supplemental disposition/repair and fresh candidate verification.

- 2026-09-09 checkpoint: Phase A/B are complete (documentation-only commit `56b9c2c`); 15 accepted repairs and their regressions are implemented. Aggregate matrix, final independent review and stable-candidate verification remain open. Frozen disk-backed Flutter passed73.54s; UI failed75.11s at the first tap, confirming AUDIT-UI-001. Initial candidate Flutter107.45s/UI146.19s and Linux Browser race10.178s pass; UI will be repeated after the final evidence-bound repair. Interim harness/race/integration runs failed while new mobile fixture sizes/API responses were being corrected. One process-preview deadline failure under concurrent integration load requires isolated recheck.


- 2026-09-09: Frozen baseline passes full repoctl, full race, Docker integration, Linux Browser native race, real Podman coexistence (126.631s), real Android concurrent leases/manual termination (48.587s), and private-clone release verification (six targets, two byte-identical builds, eight files, native Linux smoke). Frozen-master Verify 34290359477 and Browser native 34290359439 pass Windows/macOS/Linux. Flutter/UI integration is running; final candidate validation is pending.
- 2026-09-09: Two initially guessed CLI test selections used incorrect tags/names and returned “no tests to run”; they are not coverage. Correct Podman integration was then executed. Android fixture creation first requested an absent image variant; a listed installed image succeeded in a private template home.
- 2026-09-09: Phase A temporary overlays reproduce AUDIT-BOUNDARY-001 (exact Android log limit), AUDIT-DOCS-001 (fenced fake fragment target), AUDIT-REDACTION-001 (Android UI post-redaction budget), and a High readiness cleanup candidate AUDIT-CLEANUP-001. The latter full Create entry point returns released, removes its source, and retries seven times after an unconfirmed process-tree result (0.024s). Findings remain untriaged until Phase B; no product edits.

- 2026-09-09: Baseline repoctl check passed unit/vet, then failed docs-check because the supplied Japanese plan lacked translation_of and source_sha256. Corrected only plan metadata after reviewing the paired contents; this is audit setup, not a product repair. The initially assumed .github/workflows/verify.yml path does not exist; the actual workflow is ci.yml.


Known pattern to investigate repository-wide: exactly filling a documented
collection limit may be falsely marked truncated. Preserve disproven hypotheses
as such; do not rewrite them as confirmed root causes.

## Decision Log

- 2026-09-09 supplemental Phase B: all four postmerge comments reproduce against frozen031869c8. ACCEPT AUDIT-BOUNDARY-003 (128-page creation overflow), AUDIT-STATE-001 (ignored AX wait), AUDIT-STALE-002 (pressed fingerprint) and AUDIT-CLI-001 (mutation table), all Medium. The original15 remain accepted; total19. Three CDP negatives failed0.061s; native CLI mutation output failed both create/close assertions while independently verifying side effects and sibling retention. Supplemental reports own details. This is an explicit audit scope reconciliation, not a separate PR task.

- 2026-09-09 Phase B checkpoint: all bounded Phase A reviews and182 historical rows across all20 completed plans are complete; production is still frozen at031869c8. ACCEPT all15 reproduced findings (7 High,6 Medium,2 Low), as recorded in `docs/audits/repository-correctness/findings.md`. No confirmed finding is deferred/rejected/duplicated. Phase C may now modify only these contracts and meaningful regression tests. Baseline real Flutter passes with disk-backed temporary storage; real UI then reproduces AUDIT-UI-001 on its first tap. Do not mark final validation complete.

- 2026-09-09 / audit implementer: Durable reports live in docs/audits/repository-correctness/ with bilingual indexes, matrix, findings, historical corpus and escape summary plus bounded subsystem annexes. They remain durable after completion. Phase-A temporary reproductions may use isolated temporary checkouts/Go overlays and lease-owned fixtures, never modify product files or shared user resources. All current findings start untriaged until the disposition checkpoint; accept in-scope Medium/Low by default, and never defer unresolved Critical/High. No public issues or global policy changes are part of this audit. Existing real local integrations plus native CI are required where available; unavailable infrastructure is recorded, not counted as a pass. Shared preventive controls require confirmed recurrence before adoption.


- Decision: Start only after PR #10 merges and freeze that resulting master
  revision for Phase A. Rationale: Browser/CDP materially expands snapshot,
  action and limit logic. Date/Author: 2026-09-09 / maintainers.
- Decision: Phase A is review-only. Rationale: preserve reviewer independence and
  finding visibility. Date/Author: 2026-09-09 / maintainers.
- Decision: Accepted findings require regression evidence before/with the fix.
  Rationale: prove the defect and prevent recurrence. Date/Author: 2026-09-09 /
  maintainers.
- Decision: Completion does not require zero findings. Rationale: coverage,
  disposition and resolution matter more than an empty report. Date/Author:
  2026-09-09 / maintainers.
- Decision: unresolved Critical and High counts must be zero. Rationale: these
  break core ownership/cleanup/persistence/target-safety guarantees. Date/Author:
  2026-09-09 / maintainers.
- Decision: `truncated` means evidence was actually omitted. Rationale: exactly
  filling the allowed limit is still complete evidence. Date/Author: 2026-09-09 /
  maintainers.
- Decision: repeated defect classes should be promoted into reusable harness
  checks/helpers. Rationale: improve future correctness, not only current bug
  count. Date/Author: 2026-09-09 / maintainers.
- Decision: durable audit docs and this plan are bilingual. Rationale: repository
  policy. Date/Author: 2026-09-09 / maintainers.

## Outcomes & Retrospective

Completed 2026-09-09. Frozen target `031869c8b9073b8e23bc17fbc55243666a52f557`;
final production revision `f2ec634baa00af5221217dbfcd5c0aef93c624c9` on
`audit/repository-correctness`. Documentation-only Phase A commit `56b9c2c`
precedes seven coherent repair commits. The bilingual
[report index](../../audits/repository-correctness/index.md),
[ledger](../../audits/repository-correctness/findings.md),
[matrix](../../audits/repository-correctness/matrix.md) and
[historical corpus](../../audits/repository-correctness/historical-corpus.md)
preserve the complete evidence and limitations.

Reviewed20 subsystems ×14 invariants and all20 pre-audit completed plans,
with186 historical material rows (182 initial +4 late external comments).
Accepted19, resolved19: Critical0, High7, Medium10, Low2.
REJECT0, DEFER0, DUPLICATE0; unresolved Critical/High/Medium/Low are all0.
Guardrail-only follow-ups without a demonstrated present defect remain explicit
in the ledger; they are not undispositioned accepted findings.

| Detection measure | Count |
| --- | --- |
| Original independent PR review S8 | 4 |
| Original repository audit S9 | 15 |
| Frozen audit reproduction S9 | 19 |
| Earliest realistic prevention S2 | 9 |
| Earliest realistic prevention S3 | 7 |
| Earliest realistic prevention S4 | 3 |

Escape categories overlap; counts are memberships among19 current findings,
not additional defects. Historical stage-by-stage analysis remains in the corpus.

| Escape reason | Findings |
| --- | --- |
| COMPOSITION_GAP | 14 |
| NEGATIVE_FIXTURE_GAP | 10 |
| ORACLE_COUPLING | 9 |
| BOUNDARY_GAP | 7 |
| FAILURE_INJECTION_GAP | 4 |
| INVARIANT_GAP | 3 |
| CONCURRENCY_GAP | 2 |
| REVIEW_CHECKLIST_GAP | 2 |
| HELPER_ONLY | 1 |

The common failure was a correct local check not carried through its consumer:
readiness lost executor certainty; UI/DOM limits ran before final redaction;
page enumeration did not constrain creation; state production omitted a field
used by stale-action reasoning. Added controls use existing durable CommandRun
barriers, shared rendered-prose extraction and opened-file capped readers;
exact boundary/artifact tests and protocol mutation fixtures exercise real
consumers. Secret-derived fingerprints are removed when they cannot safely
support input, while ordinary editable snapshots remain actionable.

Independent reviewers found and helped correct ordinary readiness cancellation
retry, a post-action opaque fingerprint, and weak fixture reachability/byte
calibration before completion. One process-preview deadline failed amid heavy
native integrations; unchanged isolated race×10 and subsequent full native CI
pass. tmpfs emulator capacity failures were diagnosed from emulator output and
rerun on disk-backed temporary storage without weakening timeouts or assertions.
Final historical reconciliation added four real defects missed by the original
completed Browser plan; corpus completion must include external follow-ups too.

Final `repoctl check`, docs/translation checks, full race and independent reviews
pass. Verify34295144985 and Browser native34295144958 pass on exact final product
revision (Windows/macOS/Linux; native evidence distinct from cross-build).
Real Docker, Podman coexistence, Android, Flutter/UI and Browser passes are in
the index. Final private-clone release verification builds all six targets twice,
compares eight identical files and passes Linux native smoke. Later changes are
audit/product-documentation completion only; no production dependency or
architectural boundary changed, and no POSIX shell was added to the core.

For future PR review, require independent exact-limit expected values, actual
injection reachability, full consumer/durable-output assertions, paired refusal
and permitted behavior, and native evidence where semantics differ. These are
recommendations and implemented tests here, not unapproved global policy changes.
Remaining limits: direct Windows second-PID-read scheduling, Java traversal
producer tests and some release close/late-persistence injections are not fully
exhausted. Same-user hostile host/ADB and every possible native schedule remain
outside the proof. No known accepted defect is left open. No merge, public tag,
release or unrelated repository operation is part of completion.

## Context and Orientation

Read AGENTS/ARCHITECTURE/PLANS/QUALITY/RELIABILITY/SECURITY/PORTABILITY/roadmap in
both languages, current product/design indexes, completed major ExecPlans,
repoctl validators and CI workflows.

Review production packages at minimum:
`internal/config`, `internal/domain`, `internal/app`, `internal/store/sqlite`,
source packages, Compose/Podman, Android, Flutter lifecycle, Android UI,
persistent process, Browser/CDP, `internal/execx`, `internal/evidence`,
`internal/paths`, release/buildinfo/assets and CLI locations where policy leaks.

## Plan of Work

### Milestone 1 — Freeze target and baseline
After PR #10 merge, update master, record HEAD, create audit branch and record
baseline failures before fixing anything.

### Milestone 2 — Audit matrix
Build invariant x subsystem matrix with reviewed/N-A/finding-ID cells.

### Milestone 3 — Historical review replay

Before broad new finding discovery, inventory the repository's prior review
follow-ups.

Do not rely only on filenames. Search completed ExecPlans for terms and sections
such as:

```text
review
finding
thread
regression
independent review
Surprises & Discoveries
failed
caught
missed
```

Start with known dedicated plans such as:

```text
docs/exec-plans/completed/bilingual-documentation-review.md
docs/exec-plans/completed/android-emulator-review.md
docs/exec-plans/completed/android-emulator-review-2.md
docs/exec-plans/completed/compose-provider-podman-review.md
```

and include any review follow-up incorporated into a normal feature ExecPlan.

For each historical finding:

1. identify the invariant and original failure mode;
2. identify the stage where it was finally detected;
3. classify the earliest realistic preventable stage (S0-S9);
4. enumerate detection opportunities that existed before discovery;
5. explain why each relevant opportunity failed;
6. classify escape reasons;
7. locate the regression test that was added;
8. confirm the regression still exercises the current production entry point;
9. inspect whether refactoring bypassed or weakened the test;
10. search the repository for the same structural pattern;
11. identify the preventive guardrail that should catch the class earlier next time;
12. create a current audit finding only when a present defect/coverage gap exists.

Create a historical-review matrix separate from new findings so “already fixed
and still covered” is positive audit evidence rather than noise. Also produce an
escape-analysis summary grouped by earliest preventable stage and escape reason.

### Milestone 4 — Boundary audit
Search all limit/truncation/pagination/retry paths. Add limit-1/limit/limit+1
fixtures for accepted semantic-boundary defects.

### Milestone 5 — State/persistence/cleanup audit
Create saga tables for intent/effect/identity/readiness/cleanup/absence/release
and inspect/fault-inject each gap.

### Milestone 6 — Identity/stale/concurrency audit
Construct reuse/race scenarios for PID, port, names, connections, serials,
targets/nodes, sibling leases and independent SQLite writers.

### Milestone 7 — Redaction/path/portability audit
Exercise chunk/truncation/multibyte secrets and native path/process behavior.

### Milestone 8 — Report/disposition
End Phase A before product fixes. Disposition all findings with required evidence.

### Milestone 9 — Remediation
Fix highest severity first using reproduce -> regression -> minimal fix ->
focused/integration/native verification.

### Milestone 10 — Promote recurring invariants
Add shared helpers/checks only where confirmed repeated defect classes justify
them.

### Milestone 11 — Independent re-review
Read-only review of changed code, original finding locations and adjacent same-
pattern code.

### Milestone 12 — Final verification
Run repoctl/docs/race/native and current real Docker/Podman/Android/Browser/release
suites. Confirm all findings dispositioned, all ACCEPT resolved, unresolved
Critical/High zero.

## Concrete Steps

1. Merge PR #10.
2. Record post-merge master revision.
3. Create audit branch.
4. Add bilingual active plans.
5. Run baseline.
6. Build audit matrix.
7. Build and replay the historical review corpus.
8. Complete all remaining Phase A reviews.
9. Publish finding set.
10. Disposition every finding.
11. Add regressions and fixes for ACCEPT findings.
12. Promote repeated patterns where justified.
13. Independent read-only re-review.
14. Final verification matrix.
15. Confirm unresolved Critical/High zero.
16. Complete bilingual retrospective.
17. Archive plans and update links/hashes.

## Validation and Acceptance

| ID | Required behavior | Evidence |
| --- | --- | --- |
| A1 | Audit target is exact post-PR-#10 merged master and frozen during Phase A. | Frozen `031869c8`; branch and Phase A report commit `56b9c2c`. |
| A2 | Phase A completes without product-code modifications. | `git diff 031869c8 56b9c2c` contains audit documentation only; product repairs start at `dc358e1`. |
| A3 | Required subsystem x invariant matrix is complete. | `docs/audits/repository-correctness/matrix.md`: 20 subsystems × 14 invariants. |
| A4 | Every semantic collection limit affecting completeness is exact-boundary reviewed. | Matrix bounds column and mobile/process-browser/Compose-release annexes; AX/node/frame/log/byte/field limits reviewed. |
| A5 | Accepted boundary defect classes have limit-1/limit/limit+1 regression coverage. | UI rune4096±1, node1000±1, encoded1MiB±1; log2000±1; DOM encoded1MiB±1; release read cap−1/exact/+1 and Browser page127/128/129 fixtures. |
| A6 | Returned count equal to limit alone never implies truncation. | Overflow record and whole-node omission controls; exact full counts stay complete. See BOUNDARY-001 and REDACTION-001/002. |
| A7 | READY/owned/clean/absent/released decisions are audited for positive proof. | Control-plane saga/positive-proof table plus current provider annexes; CLEANUP-001, OWNERSHIP-001, LIFECYCLE-001 repaired. |
| A8 | Major external effects are audited for intent/effect/identity persistence ordering. | Current-control-plane intent/effect/result review; provider receipt/persistence tests and readiness durable attempt regression. |
| A9 | Successful effect plus persistence failure cannot silently erase effect identity. | Existing process launch receipts, application/UI/browser running rows and save-failure tests; new readiness safety tests. |
| A10 | Cleanup paths are audited for actual absence proof, not command success alone. | Destroy/GC running-row quarantine regressions, empty-ID public Down rejection and real lease sibling cleanup. |
| A11 | PID/port/project/connection/serial/page/node reuse is audited for identity confusion. | Provider identity/reuse matrices; source registration, process birth/Job, Compose engine/project, AVD serial, helper digest, Browser target/node. |
| A12 | Lock loss/cancellation cannot authorize unintended later effects. | Readiness typed cancellation regression and existing lock-loss/command-tree/UI/Browser fencing tests; candidate race passes. |
| A13 | Cross-process claims use independent process/connection tests where needed. | SQLite independent-connection/process reservations, execx native process helpers, real concurrent Docker/Android/UI leases; Podman99.951s PASS. |
| A14 | Android/Browser stale actions revalidate exact targets and reject ambiguous/truncated evidence. | UI editable positive and secret/stale refusal tests, exact helper build and recovery; CDP post-predicate navigation regression. |
| A15 | `gone`/absence waits do not treat truncated evidence as complete absence. | Existing partial/truncated Android wait and Browser AX/frame absence negatives retained; reviewed in mobile/Browser matrices. |
| A16 | Redaction/truncation covers multibyte and chunk/limit boundaries. | Evidence cross-chunk/multibyte tests plus final post-redaction UI/DOM artifact byte tests; configured secret-derived hashes removed. |
| A17 | Path review covers symlink/traversal/case/space/non-ASCII/Windows paths. | Source/paths/assets/release matrices and platform path tests; private release smoke uses Unicode state root. |
| A18 | Native process semantics are separately verified on Windows/macOS/Linux where different. | Verify34295144985 native Go1.26/1.27 ×3OS PASS; Browser34295144958 real3OS PASS on final f2ec634. |
| A19 | Release tooling is audited for source identity, deterministic bytes and validated-byte publication. | Clean final f2ec634 private-clone release-verify PASS: six targets twice, eight byte-identical files, Linux smoke; release regression suite PASS. |
| A20 | Durable mechanical claims have implementation/negative tests or corrected prose. | Fragment full docsCheck negatives/positives; UI/DOM bounds and readiness durable claims corrected in both languages; docs-check PASS. |
| A21 | Every completed review/follow-up ExecPlan is inventoried or explicitly marked not applicable to the frozen target. | historical-corpus.md inventories all20 completed English plans and PR5 external24 comments. |
| A22 | Historical review regressions are mapped to current tests/entry points and checked for stale or weakened coverage. | 186 material rows map current production/regression entry and oracle limits; historical annexes distinguish replay from source inspection. |
| A23 | Each historical defect pattern is searched across other subsystems for recurrence. | Historical/current annex recurrence searches produce19 present findings; disproven hypotheses retained. |
| A24 | Historical “why tests missed it” lessons are incorporated into the current audit method/fixtures where still relevant. | Full entry callbacks, durable run/artifact assertions, independent negative/positive fixture pairs and actual encoded boundaries added. |
| A25 | Every material historical finding has a detected stage and earliest realistic preventable stage (S0-S9). | Historical annexes classify detected/earliest stages per material row; historical-corpus aggregation. |
| A26 | Every material historical finding records concrete pre-discovery detection opportunities and why they failed. | Historical annexes record earlier opportunities, weak oracles and guardrails; no reviewer intent invented. |
| A27 | Every ACCEPT current finding has escape-reason classification and earliest preventable stage. | matrix.md maps19/19 ACCEPT findings: audit replay S9; original S9=15/S8=4; earliest S2=9, S3=7, S4=3. |
| A28 | Every recurring defect class has a preventive guardrail with evidence, or an explicit reason broader prevention is impractical. | matrix.md recurring-class table: nine controls/reuse decisions with evidence and explicit limits on generic prevention. |
| A29 | Every finding has ACCEPT/REJECT/DEFER/DUPLICATE disposition with rationale. | findings.md: ACCEPT19, REJECT0, DEFER0, DUPLICATE0; per-finding contract rationale. |
| A30 | Every ACCEPT finding has regression evidence before/with fix. | Linked Phase C annexes retain failing frozen overlays and permanent focused regressions; readiness ordinary cancellation discovered/repaired in independent review. |
| A31 | Every accepted Critical/High finding is resolved. | All seven accepted High findings repaired and independently reviewed; no Critical finding. |
| A32 | Unresolved Critical count is zero. | Zero; no Critical finding accepted or deferred. |
| A33 | Unresolved High count is zero. | Zero; High7/7 resolved. |
| A34 | Every DEFER finding has explicit risk, reason and follow-up. | No DEFER finding. Explicit non-defect coverage follow-ups and rationale remain in findings.md. |
| A35 | Repeated defect classes are promoted into reusable harness/check/test mechanisms where justified. | Shared documentProse target extraction, readiness CommandRun/runWithCancellation, opened bounded release reader, UI raw/normalized/result bound; domain-specific exact boundary oracles. |
| A36 | Independent final read-only re-review finds no unresolved accepted defect in changed/adjacent code. | Separate author/reviewer checks cover readiness/docs, Compose/release, mobile, Browser and CLI; final supplemental CDP race×5 PASS2.039s, no remaining defect. |
| A37 | Final repoctl/docs/translation/race/native/integration matrix passes. | Final repoctl/check/docs/race PASS; Verify34295144985 and Browser34295144958 PASS; real Docker/Podman/Android/Flutter/UI/release evidence and applicability in audit index. |
| A38 | Both language plans contain final statistics/evidence/retrospective before archival. | Bilingual Outcomes contain19 findings,186 corpus rows, stage/category counts, controls, exact final CI and limits; docs-check and archive links verified. |

Existing green CI alone does not satisfy the audit.

## Idempotence and Recovery

Phase A is observational. Findings/reports are versioned so interrupted review can
resume. Remediation uses small coherent commits. Do not weaken an invariant to
recover green tests. If a finding proves invalid, change it to REJECT with
evidence instead of forcing a code change.

## Artifacts and Notes

Durable audit entry points (annex links and final evidence are in index):

    docs/audits/repository-correctness/
      findings.md
      findings.ja.md
      matrix.md
      matrix.ja.md
      historical-corpus.md
      historical-corpus.ja.md
      index.md
      index.ja.md

Use stable finding IDs and no secrets. Final summary records target revision,
areas reviewed, finding statistics, new regressions/checks and final run IDs.

## Interfaces and Dependencies

No new production dependency is required. Use existing Go tests, repoctl, native
CI, fake/injected providers, real integration fixtures, SQLite failure injection,
process helpers and Browser/Android fixtures. Prefer Go/portable harness code over
audit shell scripts.

## Unresolved Issues to Settle During Milestone 1

All twelve setup questions are settled for this audit:

1. Reports live under `docs/audits/repository-correctness/`; index, findings, matrix and historical-corpus are bilingual entry points.
2. Retain those reports after completion; the archived plan links their detailed evidence.
3. Frozen reproductions use private temporary overlays/clones; only meaningful permanent regressions enter production history.
4. No Medium finding is deferred: all ten are accepted and repaired.
5. Both Low findings are accepted and repaired.
6. Required installed integrations ran: Docker, Podman coexistence, Android, Flutter/UI, native Browser, and release verification. Native OS checks use CI; cross-build alone is insufficient.
7. No unsolicited QUALITY policy change. Exact-limit semantics are enforced by the existing contract and focused regressions.
8. No noisy syntax-only limit checker: omission depends on domain eligibility, encoding and skipped records. Matrix controls record positive/negative oracles instead.
9. No DEFER findings, so no GitHub issues are created for them.
10. Smaller recurring audits remain a recommendation, not a newly imposed workflow.
11. S0–S9/escape fields are used here; making them mandatory for future plans needs a separate policy decision.
12. Concrete controls stay in existing harness/tests and the audit matrix. No new general policy document is created.
