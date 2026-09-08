---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Flutter Android applications

[日本語](flutter-android-runtime.ja.md)

A Flutter application is separate from its lease-owned `android-emulator` runtime.
Existing Compose-only and Android-only manifests require no changes.

## Manifest contract

Declare `applications.<name>` with `type: flutter-android`, a known `source`,
a known Android `runtime`, and source-relative `project_directory` (default `.`).
Declare `build.command` as a nonempty argv array: its first element selects the
Flutter executable on PATH or an explicit host path; subsequent arguments are
passed literally without shell interpretation. `build.artifact` is a nonempty
project-relative `.apk` path confined to that project. Optional `build.timeout`
is a positive duration, defaulting to `20m`. The package and launch activity are
always explicit: `package` is a dotted Android identifier; `activity` is either
a fully qualified class name or a leading-dot class relative to the package.
Activity segments start with an ASCII letter or underscore and contain only ASCII
letters, digits and underscores; nested-class `$` names are deliberately excluded
to keep device-shell commands literal.

A component selects an application with `application: <name>` and must use the
same Android runtime. An optional `reverse` array declares `device_port` in
1..65535 and `endpoint: <component>.<endpoint>`. The endpoint must belong to a
transitive dependency of every component selecting that application and use TCP
(the endpoint protocol defaults to TCP). Endpoint references must be unambiguous.
Packages and device ports must be unique across applications sharing a runtime;
separate runtimes and separate leases may reuse them.

Selected applications must not share the same source and normalized source-relative
APK output path (compared case-insensitively for portability). Several components
may select the same application. Plan/create reject that collision before builds
or runtime effects; use distinct project/output paths. Collisions between applications selected only
by separate stacks do not prevent independent use. Project paths and their ancestors
through the allocated source root must not be symlinks, even when the target stays
inside the source; this is checked before and after the build.

## Lifecycle and evidence

Planning lists selected applications, source, runtime, build artifact and reverse
requirements without building or allocating runtimes. Prerequisite diagnostics
are available through `agent-env doctor <repository> --runtime flutter-android`.
They inspect every declared application in its current source checkout, check the
selected Flutter executable, project metadata and Android AVD prerequisites, and
require no Docker or worktree allocation. They do not install SDKs or accept
licenses. Create independently validates pinned projects and selected dependencies.
Builds run in pinned disposable source worktrees,
with bounded argv execution and captured, redacted output. A regular confined
APK is hashed before installation. Evidence retains source commit, Flutter
version, build command/path/output, APK SHA-256 and target runtime/serial.
A digest identifies the installed build; it does not prove reproducibility or
promise retention of the APK after destruction.

Application reconciliation skips reverse-mapping and backend-endpoint queries
when that application declares no reverse mappings.

Creation rejects an already-installed declared package before installing, so a
package inherited from an AVD template cannot be attributed to the new APK digest.
It installs only on the confirmed lease-owned serial and verifies the package,
resolves actual selected Compose host endpoints, requires loopback TCP addresses
(remote Docker hosts are rejected), configures and verifies reverse
mappings, then launches the explicit activity. READY requires all these steps.
The application need not remain foreground or running afterward. Missing packages
or required mappings degrade a lease; ambiguous device identity quarantines it.
A requested mapping is not proven owned: if establishment was never confirmed
and a mapping exists, cleanup quarantines it without removal. An absent mapping
is safe to leave absent. Cleanup removes only confirmed owned mappings and
destroys private Emulator state
through the existing Android lifecycle. Build and failure evidence remain available.
A durable `build_unconfirmed` guard is set before the build and remains after a
crash or unconfirmed process/output termination. It blocks subsequent source
cleanup, including forced destroy after restart. Investigate termination evidence;
the CLI does not silently clear this guard.
A separate durable `build_evidence_incomplete` guard is set before the build and
cleared only after both required build-log artifacts and final lease state are
saved successfully. Failed evidence persistence keeps the lease quarantined and
retains the source/APK during normal or forced cleanup, even after the store
recovers. Investigate and recover the missing evidence before releasing this guard.

`destroy --dry-run`, including with `--force`, reports either durable build guard
as a cleanup blocker without changing the lease or removing resources.

Named tests may use `${android:<runtime>:serial}` only for a selected, confirmed
lease-owned Android runtime. Existing `${lease_id}` and `${env:NAME}` remain
supported. Flutter integration tests may rebuild/reinstall a different APK;
their results do not prove execution of the lifecycle-installed APK. Named-test
output and persisted run `notes` state this distinction. [Android UI observation](android-ui-observer.md) is a separate capability over the
owned runtime. Historical artifact promotion remains follow-up work.

## Real integration validation

With Git, Flutter and Docker Compose available on PATH, a running Docker engine,
an Android SDK configured through `ANDROID_HOME` or `ANDROID_SDK_ROOT`, a stopped
installed AVD selected by `AGENT_ENV_ANDROID_TEMPLATE`, working Emulator
acceleration, and a Flutter-compatible Java/Gradle/Android build toolchain, run:

Provide enough on-disk temporary storage for concurrent private AVD copies and
the template's userdata partitions. A memory-backed temporary directory may be
too small even when the host has ample disk space. Configure the test process's
native temporary-directory location when necessary; do not shrink templates or
weaken ownership checks to fit.

```text
go test -tags=flutterintegration -run TestRealFlutterAndroidBackendLease -v ./internal/cli -timeout=40m
```

Optional fixture-only `AGENT_ENV_FLUTTER_OFFLINE_FIXTURE=1` adds Flutter's official
`flutter create --offline` option while creating the disposable project. It uses an
existing package cache and fails if that cache is incomplete; it does not change
ordinary build/runtime behavior or relax any acceptance checks. Set it through the
host environment when cached fixture creation is needed.

This opt-in test creates a disposable Flutter project and two concurrent leases.
Each builds and installs a debug APK, launches it against its own real Compose
backend through reverse, observes the Flutter HTTP request, and runs a named
`adb get-state` test with the lease-owned serial. It destroys the first lease
without force and checks that the sibling remains READY and issues a fresh guest
HTTP request to its backend,
then destroys the sibling. Explicit selection fails
when prerequisites are absent. Normal Flutter/Gradle builds may download declared
dependencies under already accepted licenses; the fixture does not accept licenses.
Run it separately from tests that reserve real Emulator ports. Failed cleanup
retains evidence and resources for explicit investigation.
