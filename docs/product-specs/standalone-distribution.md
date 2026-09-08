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
checksums plus a release manifest. The Git tag is the sole release-version authority. A release request is valid only when `HEAD` equals the tag commit, the working tree is clean, the tag is exactly `v<semver>`, and the requested version equals the tag without its leading `v`. Development binaries report `devel` honestly.

Archives use names such as `agent-env_v0.1.0_linux_amd64.tar.gz` and contain a top-level directory such as `agent-env_v0.1.0_linux_amd64/` with the executable, `LICENSE`, and `README.txt`. Every archive file uses the tagged commit timestamp as its modification time; wall-clock time is not used.

Writable state uses the existing OS-native state root and `AGENT_ENV_HOME`
override. The executable never writes beside itself. Agent-owned immutable
assets use content-addressed, digest-verified materialization under that state
root and are created only when a capability needs them.

The first release surface is GitHub archive downloads. Signing, notarization,
package-manager recipes, SBOMs, and attestations remain follow-up work.
