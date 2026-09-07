// Package domain contains adapter-independent lease identities and state.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

var ErrLockLost = errors.New("operation ownership lost; stop effects and reconcile recorded intent")

type Source struct {
	Alias          string    `json:"alias"`
	RepositoryID   string    `json:"repository_id"`
	RepositoryPath string    `json:"repository_path"`
	RequestedRef   string    `json:"requested_ref"`
	Commit         string    `json:"resolved_commit"`
	WorktreePath   string    `json:"worktree_path"`
	CheckoutMode   string    `json:"checkout_mode"`
	Writable       bool      `json:"writable"`
	ResolvedAt     time.Time `json:"resolved_at"`
}

type Component struct {
	Name         string   `json:"name"`
	Runtime      string   `json:"runtime"`
	Services     []string `json:"services"`
	Capabilities []string `json:"capabilities"`
}

type Resource struct {
	ID         string            `json:"id"`
	LeaseID    string            `json:"lease_id"`
	Runtime    string            `json:"runtime"`
	Kind       string            `json:"kind"`
	ExternalID string            `json:"external_id"`
	Metadata   map[string]string `json:"metadata"`
}

type Runtime struct {
	LeaseID      string           `json:"lease_id"`
	Name         string           `json:"name"`
	Type         string           `json:"type"`
	Source       string           `json:"source"`
	Project      string           `json:"project"`
	Directory    string           `json:"directory"`
	Files        []string         `json:"files"`
	Services     []string         `json:"services"`
	Context      string           `json:"docker_context"`
	ConfigDigest string           `json:"config_digest"`
	ConfigPath   string           `json:"config_path"`
	Started      bool             `json:"started"`
	Android      *AndroidEmulator `json:"android,omitempty"`
}

// AndroidEmulator records a lease-owned slot, private writable AVD and process
// birth identity. Port availability and a PID alone never establish ownership.
type AndroidEmulator struct {
	Template     string `json:"template"`
	SDKPath      string `json:"sdk_path,omitempty"`
	TemplatePath string `json:"template_path,omitempty"`
	SystemImage  string `json:"system_image,omitempty"`
	AVDName      string `json:"avd_name,omitempty"`
	AVDHome      string `json:"avd_home,omitempty"`
	AVDPath      string `json:"avd_path,omitempty"`
	ConsolePort  int    `json:"console_port,omitempty"`
	ADBPort      int    `json:"adb_port,omitempty"`
	Serial       string `json:"serial,omitempty"`
	ProcessID    int    `json:"process_id,omitempty"`
	ProcessStart string `json:"process_start,omitempty"`
	State        string `json:"state,omitempty"`
}

var ErrResourceIdentity = errors.New("resource ownership is uncertain; quarantine without deleting")

type Lease struct {
	ID               string          `json:"id"`
	Owner            string          `json:"owner"`
	Purpose          string          `json:"purpose"`
	Mode             string          `json:"mode"`
	Repository       string          `json:"repository"`
	Stack            string          `json:"stack"`
	Desired          string          `json:"desired_state"`
	Observed         string          `json:"observed_state"`
	CreatedAt        time.Time       `json:"created_at"`
	HeartbeatAt      time.Time       `json:"heartbeat_at"`
	ExpiresAt        time.Time       `json:"expires_at"`
	ManifestDigest   string          `json:"manifest_digest"`
	ManifestPath     string          `json:"manifest_path"`
	ManifestCommit   string          `json:"manifest_commit"`
	ManifestModified bool            `json:"manifest_modified"`
	SourceSetDigest  string          `json:"source_set_digest"`
	Manifest         json.RawMessage `json:"manifest"`
	Sources          []Source        `json:"sources"`
	Components       []Component     `json:"components"`
	Runtimes         []Runtime       `json:"runtimes"`
	Resources        []Resource      `json:"resources"`
	Diagnostics      []string        `json:"diagnostics"`
}

type Event struct {
	ID      string    `json:"id"`
	LeaseID string    `json:"lease_id"`
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	Message string    `json:"message"`
}

type CommandRun struct {
	ID         string    `json:"id"`
	LeaseID    string    `json:"lease_id"`
	Name       string    `json:"name"`
	Argv       []string  `json:"argv"`
	Source     string    `json:"source"`
	Directory  string    `json:"directory"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	ExitCode   int       `json:"exit_code"`
	StdoutPath string    `json:"stdout_path"`
	StderrPath string    `json:"stderr_path"`
	Status     string    `json:"status"`
}

type Artifact struct {
	ID        string    `json:"id"`
	LeaseID   string    `json:"lease_id"`
	RunID     string    `json:"run_id,omitempty"`
	Kind      string    `json:"kind"`
	Path      string    `json:"path"`
	Digest    string    `json:"digest"`
	CreatedAt time.Time `json:"created_at"`
}

// SourceDigest depends only on the sorted immutable source tuple.
func SourceDigest(sources []Source) string {
	type tuple struct{ Alias, Repository, Commit string }
	items := make([]tuple, 0, len(sources))
	for _, s := range sources {
		items = append(items, tuple{s.Alias, s.RepositoryID, s.Commit})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Alias < items[j].Alias })
	b, _ := json.Marshal(items)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func Transition(l *Lease, next string) error {
	allowed := map[string][]string{
		"requested":   {"allocating", "failed", "releasing"},
		"allocating":  {"starting", "failed", "releasing", "quarantined"},
		"starting":    {"ready", "failed", "releasing", "quarantined"},
		"ready":       {"degraded", "releasing", "unknown"},
		"degraded":    {"ready", "releasing", "quarantined", "unknown"},
		"failed":      {"releasing", "quarantined"},
		"releasing":   {"released", "quarantined"},
		"quarantined": {"releasing", "degraded"},
		"released":    {"quarantined"},
		"unknown":     {"ready", "degraded", "releasing", "quarantined"},
	}
	if l.Observed == next {
		return nil
	}
	for _, value := range allowed[l.Observed] {
		if value == next {
			l.Observed = next
			return nil
		}
	}
	return fmt.Errorf("invalid lease transition %q -> %q", l.Observed, next)
}

func GCEligible(l Lease, now time.Time) bool {
	return GCEligibleWithGrace(l, now, 5*time.Minute, time.Minute)
}

// GCEligibleWithGrace protects recently expired or recently used leases. It is
// only the temporal/state gate; app also checks active runs and resource identity.
func GCEligibleWithGrace(l Lease, now time.Time, expiryGrace, heartbeatGrace time.Duration) bool {
	if expiryGrace < 0 || heartbeatGrace < 0 || l.ExpiresAt.IsZero() {
		return false
	}
	return l.Desired != "released" && !l.ExpiresAt.Add(expiryGrace).After(now) && !l.HeartbeatAt.Add(heartbeatGrace).After(now) && l.Observed != "quarantined" && l.Observed != "allocating" && l.Observed != "starting" && l.Observed != "releasing"
}
