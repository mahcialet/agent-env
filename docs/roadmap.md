---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Roadmap and unresolved decisions

Android Emulator/Flutter, browser/CDP observation, local OCI registry promotion, remote Git caching, provider PR shorthand, generic host processes, and distributed coordination are deferred. None are implemented MVP features. Start them only after core acceptance passes.

Do not block the MVP on all of these. Record decisions as ADRs when they become concrete.

## Repository harness

- Confirm the final hard limit for root `AGENTS.md`; the current decision is 150 lines with an 80–120 line target.
- Decide whether document freshness past `last_verified` is warning-only or CI-blocking after the initial MVP.
- Decide whether nested `AGENTS.md` files are ever needed; default is no for the MVP.
- Refine the first enforced package-dependency graph after the package layout exists.
- Decide whether generated CLI reference and manifest JSON Schema should join the generated DB schema in the MVP or immediately after it.
- Decide whether historical handoffs remain in the main branch indefinitely or move to a separate archival policy later.

## Repository and release metadata

- Confirm the final repository/module path; current assumption is `github.com/mahcialet/agent-env`.
- Existing MIT license is preserved; the historical handoff license question is resolved.
- Release packaging method is undecided: GitHub Releases, package managers, or both.

## Manifest trust

- Decide how base-branch/trusted manifests are selected for PR reviews.
- Define the exact overlay fields an untrusted target ref may change.
- Decide whether `--manifest-ref` is needed.

## Remote sources

- Authentication and credential passthrough for HTTPS/SSH remotes.
- Bare mirror/cache lifecycle.
- Generic ref syntax versus GitHub/GitLab PR shorthand.
- Behavior when a requested ref is force-updated during planning.

## Windows command execution

- Final handling for `.cmd` and `.bat` wrappers with arbitrary argv.
- Whether to use `golang.org/x/sys/windows` argument helpers directly.
- Future process-tree management with Job Objects for non-Compose runtimes.

## Compose isolation

- Exact policy for fixed host ports: hard failure versus generated override.
- Support and policy for external networks/volumes.
- Whether to support Podman Compose later through a separate adapter.
- Minimum supported Docker Compose v2 version.
- How much of `docker compose config` normalized output is stable enough to persist as structured evidence.

## State and concurrency

- Whether a separate cross-process host lock is needed beyond SQLite transactions and uniqueness constraints.
- Recovery semantics if the CLI is killed between an external side effect and the following database event.
- Event compaction and retention.
- Schema migration rollback policy.

## Review and fix workflows

- Branch naming and ownership for fix leases.
- Per-source writable selection in multi-repo environments.
- Whether a review lease can be forked into a fix lease while preserving a reproduction checkpoint.

## Android and browser extensions

- Emulator slot pool versus per-lease AVD data directories.
- Cross-platform Android SDK discovery.
- Windows-host Emulator control versus WSL clients.
- Flutter `.bat` execution details.
- UIAutomator/Flutter semantics snapshot format.
- CDP/browser adapter ownership and whether browser resources run inside Compose or on the host.

## Artifact reproducibility

- Local OCI registry implementation and configuration.
- Promotion policy for the final image used by a test run.
- Reference-based image retention and Registry garbage collection.
- Recording base-image digests and toolchain identity.
- Rebuild comparison versus exact-artifact replay.

## CI infrastructure

- Native Docker Compose integration coverage on macOS and Windows may require self-hosted runners.
- Android Emulator integration will require hardware acceleration and platform-specific runners.

---
