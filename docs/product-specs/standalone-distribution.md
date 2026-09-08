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
Linux amd64/arm64. Archives contain a relative top-level directory and regular files;
checksums and a release manifest accompany the six archives. The Git tag is the sole release-version authority. A release request is valid only when `HEAD` equals the tag commit, the working tree is clean, the tag is exactly `v<semver>`, and the requested version equals the tag without its leading `v`. Development binaries report `devel` honestly.

Archives use names such as `agent-env_v0.1.0_linux_amd64.tar.gz` and contain a top-level directory such as `agent-env_v0.1.0_linux_amd64/` with the executable, `LICENSE`, and `README.txt`. Every archive file uses the tagged commit timestamp as its modification time; wall-clock time is not used.

Writable state uses the existing OS-native state root and `AGENT_ENV_HOME`
override. The executable never writes beside itself. Agent-owned immutable
assets use content-addressed, digest-verified materialization under that state
root and are created only when a capability needs them.

The first release surface is GitHub archive downloads. Signing, notarization,
package-manager recipes, SBOMs, and attestations remain follow-up work.

## Release construction and validation

The accepted tag is `vMAJOR.MINOR.PATCH`, with decimal numeric components and no
leading zeros except `0` itself. Prerelease and build suffixes are not accepted.
Exactly one valid version tag must resolve to HEAD. Lightweight and annotated tags
are accepted; signature verification is outside this contract. Requested version
is the tag without `v`; no VERSION file or secondary version authority is used.
Clean means no tracked worktree changes, staged changes or non-ignored untracked
files. Ignored outputs do not invalidate the source.

```text
go run ./tools/repoctl release-build --version X.Y.Z --out <new-directory>
go run ./tools/repoctl release-check --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-repeat --dir <directory> --version X.Y.Z
go run ./tools/repoctl release-smoke --dir <directory> --version X.Y.Z
```

The build refuses existing output directories and stages privately before
publishing the complete local artifact set. Windows targets use `.zip`; darwin
and linux targets use `.tar.gz`, for both amd64 and arm64. Each archive contains
only its top-level directory, native executable, MIT `LICENSE` and `README.txt`.
README.txt is generated deterministically from bilingual text in the harness.

`checksums.txt` uses one lowercase SHA-256, two spaces, archive basename and LF per
line, sorted by filename. `release-manifest.json` uses schema version 1 and records
product/version/tag/commit, source timestamp, actual builder Go version, target
identity, archive and executable names/digests, executable build identity and
bundled asset metadata. The current release embeds no runtime companion assets,
so its asset list is empty. The optional Android UI helper remains an external
build. Static checks verify executable build information without running foreign
binaries. A passed cross-build is not native execution evidence.

Release commands also accept `--tag vX.Y.Z` instead of `--version X.Y.Z`.
CI uses `--tag-env` to read `AGENT_ENV_RELEASE_TAG` directly, without interpolating
a tag into a shell command. The final output contains exactly eight files:
six archives, `checksums.txt` and `release-manifest.json`. `release-repeat` validates
an existing tag-specific set and compares it with a fresh build. Construction uses
a private checkout of the exact commit, so ignored source files or local edits
hidden by Git index flags cannot enter the packaged executable.

## Executable asset inventory

`version --output json` includes `data.assets`, an array of objects with `name`,
`version`, `sha256` (lowercase hexadecimal), and `size` (bytes). The current
production inventory is explicitly `[]`, never omitted or null. Table output
reports `Bundled assets: 0`. External helper APKs and test fixtures are not
bundled assets. Inventory inspection does not create state or discover tools.

An absolute `AGENT_ENV_HOME` works even when no user home can be discovered.
Runtime staging files, including the verified UI helper installation copy, stay
under the resolved state root. External tools retain their documented host-owned
state, such as Git worktree registration and the shared ADB server.
