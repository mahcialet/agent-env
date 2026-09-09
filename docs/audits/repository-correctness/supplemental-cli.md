---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Supplemental Browser CLI result audit

[日本語](supplemental-cli.ja.md) · [Audit index](index.md)

## AUDIT-CLI-001: mutation tables print the inventory captured before the effect

Severity: Medium. Disposition: ACCEPT. Source: [PR10 comment3963154182](https://github.com/mahcialet/agent-env/pull/10#discussion_r3963154182), discovered during final historical reconciliation. The frozen031869c8 implementation in `internal/cli/browser.go` prints `Observation.Pages` for every operation. The adapter records that inventory before target creation/closure and separately returns the affected `Observation.Page`.

Frozen native entry reproduction (Linux Chrome, sandbox enabled) inserted assertions into `TestBrowserNativeCLI` using a temporary test overlay. It created a page via default table output, independently listed pages through JSON to determine the new ID, closed it via table, then verified the original sibling remains. Both table assertions failed: create omitted the new ID; close printed the closed ID as a page inventory row. The original JSON scenario passed. The failure is user-visible output correctness; it does not demonstrate wrong-target lifecycle effects.

Detected stage S9 (postmerge external review); earliest prevention S3 (CLI result model review), with S4 table-format integration as the missed oracle. Categories C8/C10/C11: result model/normalization, missing table negative/positive coverage, and JSON-only happy-path coverage. No command or schema change is needed.

## Repair and guardrail

Successful page-create/page-close tables render `Observation.Page` with explicit `Page created`/`Page closed` labels. Failure output retains the existing diagnostic path and never claims successful mutation. Read-only page inventory and JSON output retain their contracts. Permanent native CLI assertions traverse the executable and validate IDs against independent JSON inventory, sibling retention and actual removal. The initial permanent test failed before repair too. An initial edit referenced a nonexistent observation status member and failed compilation; success now uses the actual returned operation error. Final native race evidence is recorded in the index.
