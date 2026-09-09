---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Repository correctness audit

[日本語](index.ja.md)

Execution authority: [completed audit plan](../../exec-plans/completed/repository-correctness-audit.md).
Frozen Phase A target: `031869c8b9073b8e23bc17fbc55243666a52f557`, post-PR-#10 master. Working branch: `audit/repository-correctness`.

Phase A/B are complete. The initial15 accepted findings have passed implementation/regression/native acceptance; four postmerge findings were subsequently reproduced and accepted, for19 total. All19 are resolved, independently reviewed and validated; the audit is complete. Historical regression names do not imply that every past defective revision was mutation-replayed.

## Reports

- [Documentation history and fragment audit](documentation.md) / [日本語](documentation.ja.md)
- [Control-plane audit](current-control-plane.md) / [日本語](current-control-plane.ja.md)
- [Mobile historical corpus](history-mobile.md) / [日本語](history-mobile.ja.md)
- [Mobile current audit](current-mobile.md) / [日本語](current-mobile.ja.md)
- [Process/browser/MVP historical corpus](history-process-browser.md) / [日本語](history-process-browser.ja.md)
- [Compose/release historical corpus](history-compose-release.md) / [日本語](history-compose-release.ja.md)

- [Complete historical inventory and escape summary](historical-corpus.md) / [日本語](historical-corpus.ja.md)

- [Compose and release current audit](current-compose-release.md) / [日本語](current-compose-release.ja.md)

- [Process and Browser current audit](current-process-browser.md) / [日本語](current-process-browser.ja.md)

- [Finding disposition and remediation](findings.md) / [日本語](findings.ja.md)

- [Subsystem × invariant and guardrail matrix](matrix.md) / [日本語](matrix.ja.md)

- Late PR #10 supplements: [Browser/CDP](supplemental-browser.md), [CLI result](supplemental-cli.md).

## Baseline evidence

| Check on frozen product | Result |
| --- | --- |
| `repoctl check` | PASS after adding missing metadata to the user-supplied Japanese audit plan. The initial documentation failure is retained in the plan. |
| `go test -race ./...` | PASS; full package matrix. |
| `repoctl test-integration` | PASS real Docker and tagged suite, including process/Compose coexistence. |
| Linux Browser native with race | PASS10.130s, sandbox enabled. |
| Native frozen-master CI | Verify34290359477 and Browser native34290359439 PASS Windows/macOS/Linux. Candidate CI is still required after remediation. |
| Real Podman concurrent leases and Docker coexistence | PASS126.631s; explicit integration flags and installed podman-compose. |
| Real Android Emulator concurrent leases | PASS48.587s, independent leases and manual-termination reconciliation. Private temporary template home. |
| Flutter Android backend lease | FAIL342.51s, both builds completed but runtime readiness expired. Both emulator logs prove insufficient tmpfs space (12GiB required versus about10GiB available); not counted as accepted coverage. UI observer failed similarly332.31s. Frozen-clone retry with disk-backed temporary storage passed Flutter73.54s; UI failed75.11s at its first tap, independently confirming AUDIT-UI-001. Test limits were preserved. |
| Private-clone `release-verify` | PASS; six targets built twice, eight files byte-identical, Linux native archive smoke passed. Tags created only in private verification clones. |

Two incorrectly guessed CLI tags/test names returned `no tests to run`; they provide no coverage. Correct Podman and process integration paths are reported above. Android template setup initially requested an absent image variant, then succeeded with an installed image. No developer-specific SDK/Flutter paths are part of this report.

## Candidate verification

Production candidate: `687228645a4088547818ec07e428e5734110e18c`. This candidate validates the initial15 repairs; later Browser/CLI supplements require a fresh candidate below.

| Check | Result |
| --- | --- |
| `repoctl check` / `go test -race ./...` | PASS; full candidate suite; app race45.555s. Process preview isolated race×10 PASS8.205s without test changes. |
| `repoctl test-integration` | PASS; real Docker/Compose and process coexistence; CLI170.949s. Opt-in Podman is separately executed below. |
| Real Flutter / latest UI | Flutter107.45s PASS; latest UI139.632s PASS, including two concurrent Flutter/APK/Compose/Emulator leases and stale/privacy/input/cleanup assertions. |
| Real Android Emulator | PASS44.613s; independent leases and manually terminated sibling reconciliation. |
| Real Podman / Docker coexistence | PASS99.951s; `TestPodmanIntegrationConcurrentLeasesAndEvidence` with both opt-in flags. A first exact-name selection matched no tests and is not coverage. |
| Linux Browser native race | PASS10.178s; sandbox enabled. |
| Browser native CI34294068663 | PASS on Windows/macOS/Linux at6872286. |
| Private-clone release-verify | PASS at6872286: six targets twice, all eight files byte-identical, native Linux smoke outside source with empty PATH and Unicode state root. No public tag/release created. |

Verify34294068659: all six native Go1.26/1.27 OS jobs, five cross-builds and integration PASS. Final mobile independent review PASS (bounds race×3 4.352s; Android/helper1.280s/1.010s). Supplemental Browser work has since passed the final candidate gates below. No unexecuted case is counted as a pass.

## Final acceptance

Final production revision: `f2ec634baa00af5221217dbfcd5c0aef93c624c9`.
Full `repoctl check` and `go test -race ./...` pass; the final supplemental
protocol race suite also passes ×3 (1.564s), independently ×5 (2.039s).
Real Browser native race passes10.157s with the new default-table result assertions.
[Verify34295144985](https://github.com/mahcialet/agent-env/actions/runs/34295144985)
and [Browser native34295144958](https://github.com/mahcialet/agent-env/actions/runs/34295144958)
both PASS at this exact revision, including six native Go/OS jobs, five
cross-build jobs, Docker integration and all three real Browser OS jobs.
Private-clone release-verify at the same revision again passes six targets twice,
eight byte-identical files and native Linux smoke. Previously recorded real
Android/Flutter/UI/Podman evidence remains applicable: subsequent product changes
are confined to Browser/CDP and Browser CLI. No platform claim is inferred from
cross-build alone.

All19 accepted findings are resolved (High7, Medium10, Low2); Critical0,
REJECT/DEFER/DUPLICATE0. Corpus186, matrix20×14. All are independently reviewed.
The matrix records concrete guardrails and explicitly unexercised rare schedules;
this is bounded evidence, not universal absence of defects. Final archive/link
and translation checks accompany the documentation completion commit. No merge,
public tag or release was performed.

PR #11 follow-up is tracked in the [review ExecPlan](../../exec-plans/active/repository-correctness-review.md); original audit evidence above is retained.
