---
status: active
owner: maintainers
last_verified: 2026-09-09
---

# Browser/CDP automation

[日本語](browser-cdp-automation.ja.md)

Browser commands observe and control an explicitly declared Chromium-family
browser owned by a persistent process lease. They do not launch a second browser,
attach to an external browser, or reuse a personal profile. This implemented
contract defines setup, page selection, safe input and retained evidence.
The [process contract](persistent-process-runtime.md) defines process lifetime. Implementation and
native acceptance are tracked in the [completed ExecPlan](../exec-plans/completed/browser-cdp-automation.md).

## Manifest and prerequisites

Use a directly executable native, headless Chromium-family browser supporting the
required CDP methods. Chrome for Testing is the preferred test distribution and
is not bundled. A launcher that forks a different browser root is not supported:
CDP must identify the exact process root already owned by the lease. No browser,
Node, Python, Playwright, Selenium or ChromeDriver is needed for unrelated core
commands. Existing process executable and source-path rules still apply.

```yaml
version: 1
sources:
  app: {repository: ., default_ref: HEAD}
runtimes:
  browser-process:
    type: process
    source: app
    working_directory: .
    command:
      - chrome
      - --headless=new
      - --enable-automation
      - --user-data-dir=${runtime_dir}/profile
      - --remote-debugging-address=127.0.0.1
      - --remote-debugging-port=${port:cdp}
      - about:blank
    ports:
      cdp: {protocol: tcp}
browsers:
  web:
    type: chromium-cdp
    runtime: browser-process
    cdp_port: cdp
components:
  browser: {runtime: browser-process}
stacks:
  browser: {roots: [browser]}
```

### Binding and required switches

`browsers` is optional; when present it is a nonempty mapping. Each binding must
name an existing `process` runtime and one of its named TCP ports. A runtime has
at most one browser binding. Names must be portable and cannot collide under case
folding. A port named `cdp` never creates an implicit browser capability.

All five switches shown above are mandatory, exact singleton argv entries declared
by the target repository. The browser layer does not append them. Alternate or
duplicate protected switches, the `--` terminator, debugging pipes and profile
selection overrides are rejected. `--user-data-dir` is exactly
`${runtime_dir}/profile`; arbitrary private subpaths are not accepted. Ports and
private state remain generic process-runtime resources.

## Commands and selection

```text
agent-env create . --stack browser
agent-env browser capabilities <lease> --browser web
agent-env browser pages <lease> --browser web
agent-env browser page-create <lease> --browser web --url about:blank
agent-env browser navigate <lease> --browser web --page <page> --url http://127.0.0.1:8080/
agent-env browser snapshot <lease> --browser web --page <page>
agent-env browser dom-snapshot <lease> --browser web --page <page>
agent-env browser screenshot <lease> --browser web --page <page>
agent-env browser click <lease> --snapshot <snapshot> --node n4
agent-env browser set-text <lease> --snapshot <snapshot> --node n2 --text <replacement>
agent-env browser key <lease> --snapshot <snapshot> --node n2 --key Enter
agent-env browser scroll <lease> --snapshot <snapshot> --node n4 --delta-y 400
agent-env browser wait <lease> --browser web --wait-for text --contains Ready
agent-env browser console <lease> --browser web --duration 1s
agent-env browser network <lease> --browser web --duration 1s
agent-env browser page-close <lease> --browser web --page <page>
agent-env destroy <lease>
```

### Browser and page selection

Omitting `--browser` requires exactly one allocated browser binding. Page listings
are deterministic. Page-specific operations require an explicit page when more
than one eligible page exists; closing a page always requires its exact ID.
Creating a page defaults to `about:blank`. Navigation accepts only HTTP(S) URLs
without user credentials, or `about:blank`, and invalidates prior document handles.
Capabilities reports product/version/protocol without navigation or input.

### Lease eligibility and ownership

All browser operations hold the lease operation fence. Mutations require an
active, unexpired, ready lease. Read-only diagnostics also permit degraded leases
when native ownership can still be proved. Quarantined leases, ownership
uncertainty and unfinished command runs refuse browser operations. Every operation
revalidates the native process, exact reserved port, browser WebSocket identity,
CDP-reported root PID and private-profile command line before effects. A new
listener reusing an old port is never sufficient authority.

## Snapshots, input and limits

### Snapshot identity and supported frames

Version 1 snapshots provide structured JSON and compact semantic text. Handles
such as `n4` belong to one registered snapshot, lease, browser, page and document.
The Accessibility tree supplies roles, names and states; bounded DOM/layout
snapshots supplement that evidence. Same-origin iframe and shadow-DOM observation
are supported. Cross-origin/OOPIF observations are explicitly unsupported; all iframe semantic
input is also unsupported in this first slice. Ambiguous identities fail closed.

### Input validation and focus

Every semantic input requires `--snapshot` and `--node`. A fresh browser/page,
document loader, frame and backend DOM fingerprint must match before input.
Truncated snapshots, missing backend identity, changed nodes and cross-lease or
cross-page references are refused. There is no stored-coordinate fallback.
Unicode text uses CDP input and verified read-back. Internal fixed, target-bound
JavaScript can inspect node state; no unrestricted public JavaScript or raw-CDP
command exists. Key input supports Enter, Tab, Escape, Backspace, Delete, arrow
keys, Home, End, PageUp and PageDown.

Keyboard and text input activate the selected page before focusing the target.
The document and exact target must hold focus before dispatch, including after
select-all. Page activation or focus can have effects; a subsequent failure
without verified input outcome remains uncertain.

Explicit navigation invalidates earlier browser snapshots even for hash/history
changes. Document tokens also include a raw-URL digest (not its query text), and
node fingerprints include allowlisted nontext AX state flags. Set-text requires
an explicit `--text`; an explicit empty string clears the field. CLI validates
required operation arguments and refuses explicit zero durations before opening
the store.

### Operation limits and waits

Operations default to a 30-second timeout, with a 60-second maximum. Console and
network captures default to one second and allow 1 ms through 10 seconds. Text
replacement is valid UTF-8 without NUL, at most 4096 bytes. Scroll deltas are
bounded to ±10000 per axis. Wait predicates cover load, URL, accessible text/role
and disappearance; timeout ends polling. Response, node and artifact bounds expose
truncation or explicit failure instead of silently treating partial evidence as
complete.

URL waits require a nonempty `--contains`; `--role` is only available for text or
node-disappearance predicates. An incomplete snapshot cannot establish that a
node is gone. Input run evidence records the page, source snapshot and node before
acting, including when the browser response is lost; entered text stays redacted.

## Evidence, privacy and recovery

### Registered artifacts

Artifacts live below `leases/<lease>/artifacts/<run>/`: `run.json`, snapshot JSON
and text, optional DOM JSON and PNG. Result evidence contains bounded console and
network collections. Registered digests and lease identity protect snapshot reuse.
Screenshots require valid PNG data and retain dimensions and a digest tied to the
observed page. They may contain secrets; pixels are not automatically redacted.

### Privacy and redaction

Editable/password values are suppressed from semantic and DOM text evidence.
Input text is not clear-text action metadata. Recognized inherited secrets and
input values are redacted from structured results. URL credentials/query values
are redacted. Network records retain request identity, redacted URL, method,
status, type, timestamp and failure; headers and response bodies are never stored.
Console capture is bounded to its attachment window, with no promise of durable
history from before attachment. Page text, URLs and console messages may still
contain unrecognized secrets: treat all browser artifacts as private.

Before set-text, a registered artifact stores only length and full/prefix SHA
fingerprints. Later observations load this proof to redact entered text echoed
by page or console. Missing or corrupt proof fails closed and verification has a
CPU budget. Fingerprints are private recovery evidence, not encryption.

### Uncertain completion and cleanup

A disconnect during input is uncertainty, not proof that the action did not occur.
The command remains a cleanup barrier when completion cannot be confirmed. Do not
replay input automatically or discard the run to make destroy pass. Retain evidence
and establish the actual outcome before reviewed recovery. Destroy and GC use the
existing process-tree ownership rules; only proven absence permits profile removal.
There is no browser auto-restart, browser-owned PID cleanup or `Browser.close`
substitute for native termination. Downloads and reusable login profiles are out of
scope. Browser and Android UI remain separate contracts.

### Protocol and evidence bounds

Discovery is limited to 64 KiB, a WebSocket message to 8 MiB and an individual CDP
call to five seconds within the operation deadline. Lists allow 128 pages,
32 frames and 2048 AX/DOM nodes. Semantic and DOM JSON are each at most 1 MiB after redaction; AX strings longer than
4096 bytes are replaced wholly with `[TRUNCATED]`, marking the snapshot truncated. Truncated snapshots
cannot authorize input. Console and network each retain at most 256 records,
64 KiB of strings and 4096 bytes per string (oversized strings are wholly replaced
with `[TRUNCATED]`); overflowing the 512-event transport
buffer disconnects and fails capture. DOM evidence retains structure/layout only,
without any text, attributes or input values.

These persisted limits also apply after redaction. Capture duration starts before
subscription and domain enable; if enable cannot finish within it, capture fails.
Omitted console argument values are explicitly marked truncated.

For implementation responsibilities, read the [design](../design-docs/browser-cdp-automation.md). Future capabilities belong in the [roadmap](../roadmap.md).
