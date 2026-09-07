---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Reliability and recovery

External operations form a saga. Persist intent and reserved identities before side effects; record results immediately afterwards. On allocation failure, compensate in reverse order and retain events. A partially failed cleanup is quarantined and remains visible. Never erase evidence to make a retry look clean.

Reconciliation compares registry rows with Git worktrees, Compose project labels and recorded resource IDs. It detects missing/unhealthy resources, orphans, expiry, and resources remaining after release. Listing does bounded observation by default; cached listing is explicit. Listing never deletes orphans.

GC is a plan by default. Apply only with explicit request, matched identities, elapsed TTL/grace, no active command or recent heartbeat, and no unexpected tracked modifications. Force cleanup is explicit and still records an event and preserves evidence. Artifact retention is independent of environment resource retention.

Recovery starts with show/reconcile and retained logs. A missing Docker daemon, interrupted side effect, or ambiguous identity must not become a success. Resume only after inspecting authoritative state; do not restart an apparently slow live process based on a stale lock file.
