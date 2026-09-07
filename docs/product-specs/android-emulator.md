---
status: active
owner: maintainers
last_verified: 2026-09-07
---

# Android Emulator leases

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
distinct writable state, AVD identities, console/ADB ports and serials.

`plan . --stack android-runtime --output json` describes the requirement without
SDK discovery, reservations or runtime effects. `create` validates SDK and AVD
before reservation; Android-only stacks do not require Docker. Missing tools or
templates are prerequisite failures (exit status 3). Default boot budget is two
minutes. Ready requires the owned device's Android boot-completed property. The local ADB
server must be compatible with the selected SDK. Creation starts an absent server
separately from Emulator containment; malformed or incompatible existing servers
are refused without automatic replacement. Shared-server diagnostics are retained,
and destroying a lease never stops that shared SDK service.

`runtimes[].android` records the template, SDK/system image, private paths, unique
AVD name, console/ADB ports, serial, native process birth identity and state.
`doctor --runtime android-emulator` reports SDK and AVD prerequisites;
`doctor <lease-id>` reports live lease diagnostics. Show/list/reconcile observe
the live device; confirmed manual termination degrades an active lease.

Boot failure compensates in reverse order. Destroy verifies the unique AVD name
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
