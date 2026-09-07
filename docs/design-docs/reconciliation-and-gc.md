---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Reconciliation and garbage collection

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
