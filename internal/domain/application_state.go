package domain

// Application is the immutable declaration plus observed lifecycle evidence.
// An APK digest records the creation artifact, not a reproducible build claim.
type Application struct {
	BuildUnconfirmed bool                   `json:"build_unconfirmed,omitempty"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Source           string                 `json:"source"`
	SourceCommit     string                 `json:"source_commit"`
	Runtime          string                 `json:"runtime"`
	ProjectDirectory string                 `json:"project_directory"`
	Command          []string               `json:"command"`
	Timeout          string                 `json:"timeout,omitempty"`
	Artifact         string                 `json:"artifact"`
	Package          string                 `json:"package"`
	Activity         string                 `json:"activity"`
	State            string                 `json:"state"`
	Build            ApplicationBuildResult `json:"build"`
	InstalledDigest  string                 `json:"installed_digest,omitempty"`
	Reverse          []ReverseBinding       `json:"reverse,omitempty"`
}

type ReverseBinding struct {
	DevicePort  int    `json:"device_port"`
	Endpoint    string `json:"endpoint"`
	HostPort    int    `json:"host_port,omitempty"`
	Requested   bool   `json:"requested,omitempty"`
	Established bool   `json:"established,omitempty"`
}
