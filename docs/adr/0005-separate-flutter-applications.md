---
status: accepted
owner: maintainers
last_verified: 2026-09-08
---

# Separate Flutter applications from Android resources

[日本語](0005-separate-flutter-applications.ja.md)

## Context

Android Emulator ownership, allocation, ADB-server policy and cleanup already
work independently of Flutter. Applications add builds and installed state, but
must not replace that resource boundary or attach to arbitrary devices.

## Decision

Use `applications` and `component.application`. App coordinates an independent
Flutter build adapter and Android application operations on the existing Android
adapter. Runtime adapters cannot import one another, including Flutter/Android
and Flutter/Compose in both directions. CLI is the concrete wiring boundary.
Config/domain stay independent of concrete adapters. Application records extend
the existing serialized Lease payload; no new relational schema is necessary.

Build before runtime allocation, persist provenance and intended identities,
then install, resolve endpoints, reverse and launch on the proven owned serial.
The Android adapter retains exclusive responsibility for Emulator lifecycle and
ADB-server selection. App owns readiness, reconciliation and compensation.

## Alternatives and consequences

Folding Flutter into the Android runtime would prevent independent Emulator use
and couple build failures to expensive allocation. A generic workload plugin
framework would add abstractions beyond this vertical slice. Separate adapters
require app to coordinate more effects but keep resource ownership unchanged.
APK hashes establish build identity without promising historical artifact replay.

`repoctl arch-check` already rejects cross-runtime imports generically. Its
negative fixtures now explicitly cover the new Flutter node and nested packages,
with positive fixtures for app/domain/execx and same-adapter imports.
