---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Flutter Android lifecycle design

[日本語](flutter-android-runtime.ja.md)

The [product contract](../product-specs/flutter-android-runtime.md) defines the
manifest. This design explains how app coordinates Flutter builds and Android
application effects while keeping Emulator ownership independent.

## Responsibilities

Config strictly decodes the manifest. App selects applications, builds pinned
sources, records evidence and coordinates effect ordering and compensation.
The Flutter adapter discovers the CLI and executes builds. `app.FlutterProvider.Validate` checks project confinement and
metadata before app records a build intention; the adapter also validates at the
build boundary.

Repository doctor validates all current source checkouts without
Docker; create validates pinned projects separately.

Android application effects
reuse the Android adapter's confirmed
identity and compatible local ADB-server policy. They do not start an independent
ADB server or own Emulator allocation.

Domain and SQLite retain application/build
and reverse identity independently of Flutter processes. Application records extend
the existing serialized lease payload; the existing SQLite lease persistence saves
them atomically with lease state, without a new SQL table or migration.

## Ordering and ownership

The saga materializes immutable source worktrees, validates and builds selected
applications before expensive runtime allocation where possible, creates selected
Compose and Android resources, installs the verified APK, resolves actual host
endpoints, configures reverse mappings, verifies them and launches the activity.
Before effects, app rejects selected applications sharing a source and normalized
source-relative APK output path, compared case-insensitively for portability.
Multiple components may still select the same application. Build-first ordering
would otherwise let a later build overwrite an earlier application's install input.
Separate stack selections
remain independent; no additional artifact snapshot lifecycle is introduced.
Reject a preexisting declared package before installation, then verify the newly
installed package. Resolve only loopback TCP endpoints: ADB reverse targets the
local server host and cannot silently substitute a remote Docker host.
Persist each effect's intended identity before dependent effects. A mapping records
application, runtime, serial, endpoint, device TCP port and resolved host port.
The same device port is safe across separate Emulator loopback namespaces.
Application identity includes `launch_confirmed`, persisted only after a successful
activity launch. Launch intent alone cannot establish READY after a crash.
Observation requires confirmed build termination and launch, plus executable and
project working-directory consistency with the recorded build identity.

### Build termination and evidence are separate guards

A build intention persists `build_unconfirmed` before starting the process.
A crash or unconfirmed process/output termination keeps this guard set across
restart and blocks source cleanup even with force. Durable-write errors stop
cleanup; an initial quarantine alone is insufficient to protect a still-live build.
The same pre-build intent also sets `build_evidence_incomplete`. Clear this second
guard only after both required log artifacts and final lease state persist.
Confirmed process exit is not proof that required evidence was saved. A failed
artifact write keeps this durable guard across store recovery; reconciliation
quarantines and normal/forced cleanup retains source and APK until evidence is
investigated and recovered.

### Compensation preserves uncertain resources

On failure, remove proven owned mappings before existing runtime compensation.
A requested but never-confirmed reverse mapping cannot be removed when present;
retain it and quarantine. Absence is safe and permits cleanup to continue.
Confirmed destruction of the private Emulator state removes installed packages;
no global uninstall step is needed. Ambiguous identity retains reservations,
source state and evidence in quarantine. Source cleanup retains the existing
tracked-dirty rules even when a Flutter build modified tracked project files.

### Observation and test provenance

Observation verifies Android ownership, package presence, recorded mappings and
build identity consistency. It never requires a foreground activity. Named-test
serial interpolation uses observed lease identity rather than ambient selection.
Test evidence distinguishes commands that rebuild/reinstall from black-box tests
of the creation APK; both stdout and persisted run notes carry the distinction.
APK hashes are provenance, not reproducible-build claims.

## Portable execution and retained evidence

Use execx argument arrays, cancellation and native wrapper handling. The selected
first build argv element is also the executable used for Flutter version discovery.
Paths are checked lexically at manifest time and against real pinned project paths
at runtime; project paths and ancestors through the allocated source root reject
symlinks before and after builds, including links with in-tree targets. Host
ancestors above the allocated root remain outside this check. Escaped/symlink
artifacts and nonregular outputs fail. A missing
executable, invalid project or missing APK is a prerequisite/build failure, never
a reason to download a toolchain automatically. Builds inherit the existing
portable execution environment; no custom application environment map is added.
Redaction applies to retained argv and output; caches and inherited environment
values are not copied into metadata. Build timeout defaults to twenty minutes.

Native fake-adapter tests on Windows, macOS and Linux establish process and path
behavior. Real Flutter, Compose and accelerated Emulator evidence is separate.
An explicitly selected real integration test fails if prerequisites are absent.
UI automation and promoted APK retention are outside this boundary.
