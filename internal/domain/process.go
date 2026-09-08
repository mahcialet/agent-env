package domain

// PersistentProcess records immutable launch inputs and durable native ownership.
// Env and Command retain references, never expanded host credential values.
type PersistentProcess struct {
	WorkingDirectory  string            `json:"working_directory,omitempty"`
	Command           []string          `json:"command,omitempty"`
	Env               map[string]string `json:"env,omitempty"`
	Ports             map[string]int    `json:"ports,omitempty"`
	SourceCommit      string            `json:"source_commit,omitempty"`
	Directory         string            `json:"directory,omitempty"`
	StateDirectory    string            `json:"state_directory,omitempty"`
	StdoutPath        string            `json:"stdout_path,omitempty"`
	StderrPath        string            `json:"stderr_path,omitempty"`
	Executable        string            `json:"executable,omitempty"`
	ExecutableOrigin  string            `json:"executable_origin,omitempty"`
	ExecutableDigest  string            `json:"executable_digest,omitempty"`
	ResolvedDirectory string            `json:"resolved_directory,omitempty"`
	ProcessID         int               `json:"process_id,omitempty"`
	ProcessStart      string            `json:"process_start,omitempty"`
	State             string            `json:"state,omitempty"`
}
