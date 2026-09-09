---
plan_id: EP-MHOST-001
plan_type: human-validation
status: paused
owner: maintainers
last_verified: 2026-09-09
parent: EP-OPS-001
priority: 100
base_branch: master
branch: validate/ep-mhost-001
merge_policy: manual
workstreams: [multi-host-validation]
conflicts: [multi-host-runtime]
execution_mode: human-kick
pause_reason: Awaiting provisioned multi-host environment and explicit human kick
resume_when: Maintainer provisions the required environment and explicitly authorizes a validation invocation
---

# Validate the delivered multi-host product

[日本語](multi-host-human-validation.ja.md)

## Purpose / Big Picture

Exercise the already delivered multi-host product with a human-prepared environment.
This child of `EP-OPS-001` supplies new acceptance evidence, not a reimplementation
of the [completed multi-host work](../completed/multi-host-control-plane.md).

## Progress

- [x] 2026-09-09: Define stable scenarios and an explicit human-kick checkpoint.
- [ ] Provision hosts, credentials, runtimes and an approved test workspace.
- [ ] Explicitly kick preflight and resolve every BLOCKED prerequisite.
- [ ] Run all eight scenarios and retain human-readable and structured evidence.
- [ ] Track findings in review plans and rerun affected scenarios.
- [ ] Reconcile acceptance, merge evidence and retrospective before archival.

## Surprises & Discoveries

No live environment has been probed. Existing multi-host implementation was already
merged, so this plan validates the delivered behavior without repeating that work.

## Decision Log

- 2026-09-09 / implementation: Keep this plan paused until environment preparation
  and explicit human authorization. Parent hierarchy adds no dependency: depending
  on the parent's completion would prevent its human-acceptance milestone.
- 2026-09-09 / implementation: Preflight READY proves only executable discovery
  and TCP reachability, not authentication, runtime health or scenario PASS.

## Outcomes & Retrospective

Pending. No human scenario has run or passed. Record actual environment support,
scenario results, evidence usefulness and unresolved limitations after execution.

## Context and Orientation

Prepare a controller and two independently registered worker environments, valid
scoped credentials, approved source repositories and test manifests. Provide
Compose tooling, Android SDK/available AVDs, a persistent-process workload and a
supported browser on the workers used for the corresponding scenarios. Local
preflight requires `agent-env`, `git`, and `adb`; it does not inspect worker runtime
installation. Use disposable test resources and retain conservative cleanup evidence.
Do not place credentials, host-specific paths or raw sensitive logs in this plan.

## Plan of Work

A human provisions and approves the environment, then explicitly kicks each
invocation. Preflight produces READY/BLOCKED. A human performs the scenario actions
and records PASS/FINDING/BLOCKED with meaningful observations and evidence references.
A FINDING requires an existing tracked draft/active review Plan ID; BLOCKED requires
updating this paused plan. These commands never auto-run scenarios or change states.

## Concrete Steps

Create a private config JSON with `endpoints` mapping `controller`, `worker-a` and
`worker-b` to approved TCP `host:port` values. Keep credentials outside this config.
Use a new evidence directory under an existing private parent for each invocation.

```text
go run ./tools/repoctl plans human preflight --plan EP-MHOST-001 --kick --config <private-config.json> --evidence-dir <new-bundle-dir>
go run ./tools/repoctl plans human record --plan EP-MHOST-001 --kick --scenario EP-MHOST-001-01 --result PASS --evidence <observation.json> --evidence-dir <new-bundle-dir>
```

Observation JSON contains nonempty `observation` and a nonempty `evidence_refs` list.
Retain that file and its referenced artifacts privately; the bundle records its
SHA-256, not its potentially sensitive contents. For FINDING add
`--follow-up-plan <review-plan-ID>` after creating a bilingual tracked review plan.
A recorded PASS is an operator attestation, not independent automated verification.

## Validation and Acceptance

The [structured scenario contract](../validation/EP-MHOST-001.json) defines exact
scenario IDs, actions, expected observations and PASS criteria. All remain pending.

| Scenario | Action and expected observation |
| --- | --- |
| `EP-MHOST-001-01` — Two agents | Run two agents against one controller with separate worker registrations. Each assigned lease stays on its selected worker; identities and artifacts do not cross. |
| `EP-MHOST-001-02` — Authentication and workspace confinement | Attempt a missing-token request and a source path outside the approved workspace. Both requests fail without creating a lease or exposing credentials. |
| `EP-MHOST-001-03` — Compose runtime | Create, inspect and destroy a Compose lease on each worker. Explicit project ownership is isolated and cleanup removes only owned resources. |
| `EP-MHOST-001-04` — Android runtime | Allocate an available Emulator lease and observe readiness before destroy. AVD and ports are exclusively reserved, and cleanup records verified absence. |
| `EP-MHOST-001-05` — Persistent process runtime | Start a process lease, collect logs and destroy it. Process identity and descendants are observed, terminated and recorded. |
| `EP-MHOST-001-06` — Browser runtime | Use a process-backed browser lease to list pages, capture a snapshot and perform a supported action. Registered browser identity and snapshot provenance remain valid across remote transport. |
| `EP-MHOST-001-07` — TCP forwarding | Open a TCP forward to an owned lease endpoint, transfer test data and close the lease. Data reaches only the owned endpoint; forward listeners disappear on cleanup. |
| `EP-MHOST-001-08` — Recovery and redaction | Restart the controller, disconnect and reconnect a worker, retry the same operation ID, and inspect sanitized evidence. Receipts prevent duplicate effects, ambiguous resources remain visible, and secrets are absent from published evidence. |

PASS requires retained evidence supporting every expected observation. Unexpected
behavior is FINDING; absent prerequisites are BLOCKED, never PASS. Evidence bundles
contain `evidence.json` and `evidence.md`. Record the result and follow-up Plan ID
in Progress; the command's action_required does not replace that update.

## Idempotence and Recovery

Never reuse an evidence directory: existing bundles are not overwritten. If writing
is interrupted, preserve the partial bundle and use a new directory after resolving
the failure. Reread runtime state before retrying any operation. Do not forcibly
clean ambiguous resources. A previous kick does not authorize another invocation.

## Artifacts and Notes

Keep bounded sanitized artifacts outside the repository unless explicitly approved.
Bundles omit endpoint addresses, credentials, raw observations and error details.
They retain stable scenario IDs, result, timestamp, observation digest and required
follow-up. No actual host values belong in durable documentation.

## Interfaces and Dependencies

The harness owns preflight and evidence formatting; `agent-env` continues to own
product behavior. This plan has parent `EP-OPS-001` and no execution dependency on
it. Graph selection never automatically runs human-validation plans. Completion
requires this plan's own acceptance, retrospective, merge proof and bilingual move.

For Android scenario `EP-MHOST-001-04`, provision SDK/emulator/ADB on the designated
worker. The client needs `agent-env` and `git`, not local Android tooling. Create
the Emulator lease through the client, capture worker/lease identity and
worker-side ADB boot readiness, then destroy it and verify absence on that worker.
Missing worker-side Android tools are BLOCKED; client-local ADB presence or TCP
preflight alone cannot satisfy the scenario.
