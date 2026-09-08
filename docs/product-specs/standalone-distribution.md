---
status: active
owner: maintainers
last_verified: 2026-09-08
---

# Standalone distribution

[日本語](standalone-distribution.ja.md)

The standalone distribution is a native `agent-env` executable plus a license
and concise installation text. Running `agent-env version`, `--help`, and core
diagnostics requires no Go installation, repository checkout, shell, Docker,
Android SDK, Flutter, Java, Python, or Node.js. Capability commands retain
their documented external prerequisites and report them lazily.

Release archives are produced and validated by the Go repository harness. They
use the fixed six-target matrix: Windows amd64/arm64, macOS amd64/arm64, and
Linux amd64/arm64. Archives contain only relative regular files and include
checksums plus a release manifest. Version and source identity are supplied by
release build metadata; development binaries report `devel` honestly.

Writable state uses the existing OS-native state root and `AGENT_ENV_HOME`
override. The executable never writes beside itself. Agent-owned immutable
assets use content-addressed, digest-verified materialization under that state
root and are created only when a capability needs them.

The first release surface is GitHub archive downloads. Signing, notarization,
package-manager recipes, SBOMs, and attestations remain follow-up work.
