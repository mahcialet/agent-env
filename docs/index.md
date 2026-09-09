---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Find the right documentation

[日本語](index.ja.md)

Choose the question you need to answer. The [README](../README.md) introduces
agent-env and the first lease workflow. Current specifications describe behavior;
completed Plans and audit reports record what was tested at a particular revision.

## Use a capability

The [product specification index](product-specs/index.md) covers every current
contract. Start with the [manifest](product-specs/manifest-v1.md) to configure a
repository and the [CLI contract](product-specs/cli-contract.md) to operate it.

| Reader question | Read next |
| --- | --- |
| Which container engine can I select? | [Compose providers](product-specs/compose-providers.md) |
| How do I keep a native server running? | [Persistent process leases](product-specs/persistent-process-runtime.md) |
| How do I run an Emulator or a Flutter app? | [Android Emulator](product-specs/android-emulator.md), then [Flutter applications](product-specs/flutter-android-runtime.md) |
| How do I observe or interact with UI? | [Android UI](product-specs/android-ui-observer.md) or [Browser/CDP](product-specs/browser-cdp-automation.md) |
| How do I install the executable or construct a release? | [Standalone distribution](product-specs/standalone-distribution.md) |
| How do I enroll hosts and run remotely? | [Multi-host setup and operations](product-specs/multi-host-control-plane.md) |
| What was in the original MVP? | [Original MVP scope](product-specs/agent-env-mvp.md) |

## Change the repository

Start with [AGENTS.md](../AGENTS.md) for the workflow and [Architecture](../ARCHITECTURE.md)
for dependency boundaries. [Plan policy](PLANS.md) explains how active ExecPlans
control substantial work and when they can be archived.

| What you need to decide | Policy or mechanism |
| --- | --- |
| Which checks establish acceptance? | [Quality](QUALITY.md) |
| What happens after failure or interrupted cleanup? | [Reliability](RELIABILITY.md) |
| Which repositories and effects are trusted? | [Security](SECURITY.md) |
| Which OS assumptions and prerequisites are supported? | [Portability](PORTABILITY.md) |
| How should English and Japanese be maintained? | [Language policy](design-docs/bilingual-documentation.md) |
| Why does a subsystem work this way? | [Design index](design-docs/index.md) and [accepted ADRs](adr/index.md) |
| Where is the database structure defined? | [Generated schema](generated/db-schema.md), derived from migrations |

The [roadmap](roadmap.md) distinguishes implemented capabilities, current work,
and deferred decisions. It does not promise that proposed features are commands.

## Find evidence without treating history as current policy

The [quality guide](QUALITY.md) is the entry point for commands, verification
scope, and native acceptance evidence. Capability contracts and designs link to
their completed implementation Plans. For example:

- [MVP implementation](exec-plans/completed/agent-env-mvp.md),
  [Android implementation](exec-plans/completed/android-emulator-lease.md), and
  [Flutter implementation](exec-plans/completed/flutter-android-runtime.md)
  explain the delivered local foundations.
- [Browser implementation](exec-plans/completed/browser-cdp-automation.md) and
  [multi-host implementation](exec-plans/completed/multi-host-control-plane.md)
  record later acceptance. [Multi-host quality evidence](QUALITY.md#multi-host-native-verification)
  distinguishes same-runner TLS tests from unverified physical-host coverage.
- The [correctness audit index](audits/repository-correctness/index.md) separates
  frozen baseline, repaired candidates, and findings. [Historical references](references/index.md)
  explain older provenance and archive exceptions.

Designs, specifications, ADRs, and Plans carry `status`, `owner`, and
`last_verified`. Local indexes keep durable documents reachable. A review date
is not proof that a planned feature is implemented. Generated documents and
historical reference archives follow the explicit exceptions in the language
policy and references index.
