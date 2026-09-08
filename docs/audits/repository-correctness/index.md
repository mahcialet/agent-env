---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Repository correctness audit

[日本語](index.ja.md)

Execution authority: [active audit plan](../../exec-plans/active/repository-correctness-audit.md).
Frozen Phase A target: `031869c8b9073b8e23bc17fbc55243666a52f557`, post-PR-#10 master. Working branch: `audit/repository-correctness`.

The audit is in progress. Findings below are observations and proposed repairs until the disposition checkpoint; no production changes have been made. Historical regression names do not imply that every past defective revision was mutation-replayed.

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
| Flutter Android backend lease | FAIL342.51s, both builds completed but runtime readiness expired. Both emulator logs prove insufficient tmpfs space (12GiB required versus about10GiB available); not counted as accepted coverage. UI observer failed similarly332.31s. Baseline rerun uses the frozen private clone and disk-backed temporary storage, preserving test limits. |
| Private-clone `release-verify` | PASS; six targets built twice, eight files byte-identical, Linux native archive smoke passed. Tags created only in private verification clones. |

Two incorrectly guessed CLI tags/test names returned `no tests to run`; they provide no coverage. Correct Podman and process integration paths are reported above. Android template setup initially requested an absent image variant, then succeeded with an installed image. No developer-specific SDK/Flutter paths are part of this report.
