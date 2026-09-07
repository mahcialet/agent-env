package domain

// ApplicationBuildResult describes a local build, not a reproducible or retained
// APK archive. Callers redact stdout/stderr before persisting them as evidence.
type ApplicationBuildResult struct {
	Executable   string `json:"executable"`
	Version      string `json:"version"`
	Directory    string `json:"directory"`
	ArtifactPath string `json:"artifact_path"`
	Digest       string `json:"sha256"`
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
}
