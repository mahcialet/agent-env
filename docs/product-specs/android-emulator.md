---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Android Emulator leases

[日本語](android-emulator.ja.md)

An `android-emulator` runtime owns one Emulator and private writable AVD state,
independently of Flutter. APK installation, builds, adb reverse, UI interaction,
screenshots, logcat, physical devices and remote hosts remain outside this scope.

```yaml
version: 1
sources:
  app: {repository: ., default_ref: HEAD}
runtimes:
  phone: {type: android-emulator, source: app, avd: Pixel_API_35}
components:
  device: {runtime: phone}
stacks:
  android-runtime: {roots: [device]}
```

`source` identifies a pinned managed source. `avd` selects an existing local AVD
template. Android runtimes reject Compose files, project directories, services
and endpoints. Compose and Android may coexist in one stack. Components sharing
one Android runtime share that lease's Emulator; different leases always receive
distinct writable state, AVD identities, console/ADB ports and serials. Android
runtime names must also be distinct under case folding on every OS. Generated
runtime directories must not overlap any selected template or system-image tree,
including aliases through existing symlinks.

`plan . --stack android-runtime --output json` describes the requirement without
SDK discovery, reservations or runtime effects. `create` validates runnable SDK tools, host acceleration, and AVD
before reservation; Android-only stacks do not require Docker. Missing tools or
templates are prerequisite failures (exit status 3). Installed image ABI metadata
(`source.properties`) must identify an accelerated native-host-compatible image;
template ABI/CPU settings must agree. Existing shared ADB servers are checked
before reservation; an absent server is allowed and started later. Default boot budget is two
minutes. Ready requires the owned device's Android boot-completed property. The local ADB
server must be compatible with the selected SDK. Creation starts an absent server
separately from Emulator containment; malformed or incompatible existing servers
are refused without automatic replacement. Shared-server diagnostics are retained,
and destroying a lease never stops that shared SDK service.

`runtimes[].android` records the template, SDK/system image, private paths, unique
AVD name, console/ADB ports, serial, native process birth identity and state.
`doctor --runtime android-emulator` reports SDK and AVD prerequisites;
`doctor <lease-id>` reports live lease diagnostics. With a repository argument,
Android doctor validates each declared Android runtime's selected AVD, including
its template, image and host prerequisites. `logs <lease-id>` exposes retained
Emulator and shared-ADB stdout/stderr for active or released leases, with
`--component` selection and secret redaction; these are process logs, not logcat. Show/list/reconcile observe
the live device; confirmed manual termination degrades an active lease.

Boot failure compensates in reverse order. A runtime cleanup failure retains its
evidence and reservation while independent runtimes are still cleaned up.
Sources remain until all runtime cleanup is confirmed. Operation cancellation or
loss of the registry fence stops further effects. Destroy verifies the unique AVD name
and requests stop on the same authenticated console connection, then confirms
termination before deleting private writable state. Ambiguous process/AVD
identity, changed ownership evidence, unsafe paths or incomplete cleanup
quarantine the lease and retain reservations. Force never bypasses this barrier.
Repeated destroy cannot affect a new user of released ports. Template and sibling
state are never deleted. An externally occupied reserved port fails creation with compensation; allocation does not silently move to a different port. Global adb shutdown or Emulator pruning is forbidden.

SDK images and compatible host acceleration are external prerequisites. Native
fake-adapter and process tests cover Windows/macOS/Linux; actual Emulator evidence
is recorded separately in the [ExecPlan](../exec-plans/completed/android-emulator-lease.md).
Missing SDKs and cross-builds are never reported as real Emulator validation.
See the [design](../design-docs/android-emulator.md) for recovery and reservations.

## Real integration verification

Install a compatible SDK image and create a stopped AVD template. Set
`AGENT_ENV_ANDROID_TEMPLATE` to its name and SDK/AVD discovery variables as needed,
then run this native Go command:

```text
go test -tags=androidintegration ./internal/runtime/android -run ^TestRealAndroidEmulatorLeases$ -v -count=1
```

The opt-in test requires free reserved Emulator ports and usable acceleration;
missing prerequisites fail rather than skip. It starts two real Emulators with
real SQLite/app orchestration and a synthetic source provider, checks sibling
survival and manual termination, then removes only confirmed owned resources.
Uncertain cleanup retains the temporary state path printed by the test for recovery.
It does not replace the separate Git/Compose integration fixtures.

## Private Emulator helper state

Emulator child processes receive private temporary and netsim discovery paths
inside their lease's Android state. This isolates helper daemons across leases
without changing the host environment or the shared ADB-server policy. Each
instance requests a dynamic HCI port; the auxiliary netsim web UI is disabled to
avoid its fixed host port. Radio simulation and guest networking remain enabled.
Private helper processes are still part of the existing native process ownership
and confirmed-cleanup checks; a surviving ambiguous group remains quarantined.
