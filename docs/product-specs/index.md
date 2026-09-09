---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Product specifications

[日本語](index.ja.md)

Use these specifications to find user-visible behavior, supported inputs and
failure conditions. They describe current contracts unless explicitly identified
as the original MVP baseline. Implementation mechanisms belong in the
[design index](../design-docs/index.md); verification records belong in linked
ExecPlans and [quality guidance](../QUALITY.md).

## Define and operate a lease

| Reader question | Contract |
| --- | --- |
| Which commands, outputs and exit codes can I rely on? | [CLI contract](cli-contract.md) / [日本語](cli-contract.ja.md) |
| How do I declare sources, components and a stack? | [Manifest v1](manifest-v1.md) / [日本語](manifest-v1.ja.md) |
| What did the original vertical slice require? | [MVP baseline](agent-env-mvp.md) / [日本語](agent-env-mvp.ja.md) |

## Choose a runtime or application

| Capability | Contract |
| --- | --- |
| Containers with Docker or Podman | [Compose providers](compose-providers.md) / [日本語](compose-providers.ja.md) |
| A foreground native service with private state and ports | [Persistent processes](persistent-process-runtime.md) / [日本語](persistent-process-runtime.ja.md) |
| A private Android Emulator | [Android Emulator](android-emulator.md) / [日本語](android-emulator.ja.md) |
| Build, install and launch a Flutter Android APK | [Flutter applications](flutter-android-runtime.md) / [日本語](flutter-android-runtime.ja.md) |

## Observe and interact

- [Android UI observation and input](android-ui-observer.md) / [日本語](android-ui-observer.ja.md): owned Emulators, snapshots, input, privacy and recovery.
- [Browser/CDP automation](browser-cdp-automation.md) / [日本語](browser-cdp-automation.ja.md): explicit browser bindings, pages, stale references and evidence limits.

## Install or operate across hosts

- [Standalone distribution](standalone-distribution.md) / [日本語](standalone-distribution.ja.md): prerequisites, archive layout and release validation.
- [Multi-host control plane](multi-host-control-plane.md) / [日本語](multi-host-control-plane.ja.md): remote setup, placement, authentication and uncertain operations.

For deferred or unsupported capabilities, read the [roadmap](../roadmap.md).
