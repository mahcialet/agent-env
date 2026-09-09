---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Reconciliation and garbage collection

[日本語](reconciliation-and-gc.ja.md)

Reconciliation explains differences between recorded intent and external state.
Local garbage collection decides whether a lease can be cleaned up, then uses the
normal ownership and operation-lock checks to perform that cleanup. This document
separates diagnosis, candidate selection and the barriers that still block deletion.
See [lease lifecycle](lease-control-plane.md) for the underlying state model.

## Reconciliation rules

The reconciler compares desired records with observed resources and produces diagnostics/events rather than directly mutating everything it finds.

Examples:

```text
lease ready + project absent        -> degraded
lease active + worktree absent      -> degraded
lease released + project present    -> cleanup_failed/quarantined
no lease + project with agent label -> orphaned external resource
expired active lease                -> stale candidate
```

Where possible, generated resources should carry an `agent-env` label in addition to Compose’s project labels.

## GC rules

GC candidates:

- lease expired beyond grace period;
- no active command run;
- no recent heartbeat;
- resource identities match registry records;
- worktrees have no unexpected tracked modifications.

Default `agent-env gc` prints the proposed actions. `--apply` performs them.

Artifacts have independent retention from active runtime resources. Do not delete evidence merely because containers and worktrees were released.

---

## Implemented safety gates

Default policy values are five minutes of expiry grace and one minute of heartbeat grace. Domain eligibility checks both, along with protected lifecycle states. The app rejects durable `running` command records during preview and repeats that check under the operation lock before apply. Expired or unfinished work is visible through reconciliation diagnostics, and no incomplete command row is automatically declared finished.

### Cancellation does not prove completion

Destroy requests cancellation only for exact active named-run IDs and waits up to ten seconds. The named-run holder confirms descendant termination and finalizes evidence before publishing a terminal run status. An unconfirmed-termination error or evidence-finalization failure leaves the registry run `running`. Cleanup requires the lease lock and confirmed command completion; stale running records block force as well. See [reliability](../RELIABILITY.md) for timing and recovery limitations.
