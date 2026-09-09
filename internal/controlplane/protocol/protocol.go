// Package protocol defines the versioned host-runtime-independent wire contract.
package protocol

import "encoding/json"

const Version = 1
const RoleClient = "client"
const RoleWorker = "worker"

type Info struct {
	ControllerID    string `json:"controller_id"`
	ProtocolVersion int    `json:"protocol_version"`
	ProductVersion  string `json:"product_version"`
}
type Blob struct {
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}
type Capacity struct {
	MaxLeases    int `json:"max_leases"`
	AndroidSlots int `json:"android_slots"`
}
type WorkerIdentity struct {
	ControllerID   string `json:"controller_id"`
	HostID         string `json:"host_id"`
	HostInstanceID string `json:"host_instance_id"`
	Incarnation    string `json:"incarnation"`
}
type RegisterRequest struct {
	WorkerIdentity
	ProtocolVersion int      `json:"protocol_version"`
	ProductVersion  string   `json:"product_version"`
	OS              string   `json:"os"`
	Arch            string   `json:"arch"`
	Capabilities    []string `json:"capabilities"`
	Capacity        Capacity `json:"capacity"`
}
type RegisterResult struct {
	Info
	Host Host `json:"host"`
}
type Host struct {
	WorkerIdentity
	ProductVersion string   `json:"product_version"`
	OS             string   `json:"os"`
	Arch           string   `json:"arch"`
	Capabilities   []string `json:"capabilities"`
	Capacity       Capacity `json:"capacity"`
	Draining       bool     `json:"draining"`
	Removed        bool     `json:"removed"`
	Online         bool     `json:"online"`
	LastSeen       int64    `json:"last_seen"`
}
type PollRequest struct {
	WorkerIdentity
	WaitSeconds int `json:"wait_seconds"`
}
type PollResult struct {
	Operation *Operation `json:"operation,omitempty"`
}
type Source struct {
	Alias        string `json:"alias"`
	Commit       string `json:"commit"`
	BundleDigest string `json:"bundle_digest"`
}
type CreateRequest struct {
	ControlBlobDigest    string          `json:"control_blob_digest"`
	Options              json.RawMessage `json:"options,omitempty"`
	OperationID          string          `json:"operation_id"`
	Stack                string          `json:"stack"`
	ManifestDigest       string          `json:"manifest_digest"`
	PlanDigest           string          `json:"plan_digest"`
	SourceSetDigest      string          `json:"source_set_digest"`
	RepositoryID         string          `json:"repository_id"`
	HostID               string          `json:"host_id,omitempty"`
	Manifest             json.RawMessage `json:"manifest"`
	Package              json.RawMessage `json:"package"`
	Sources              []Source        `json:"sources,omitempty"`
	RequiredCapabilities []string        `json:"required_capabilities"`
	AndroidSlots         int             `json:"android_slots"`
}
type SubmitRequest struct {
	OperationID string          `json:"operation_id"`
	LeaseID     string          `json:"lease_id"`
	Kind        string          `json:"kind"`
	Payload     json.RawMessage `json:"payload"`
}
type Operation struct {
	ID             string          `json:"operation_id"`
	ControllerID   string          `json:"controller_id"`
	LeaseID        string          `json:"lease_id"`
	HostID         string          `json:"host_id"`
	HostInstanceID string          `json:"host_instance_id"`
	Epoch          int64           `json:"epoch"`
	Kind           string          `json:"kind"`
	Payload        json.RawMessage `json:"payload"`
	State          string          `json:"state"`
	Result         *Result         `json:"result,omitempty"`
}
type Result struct {
	WorkerIdentity
	OperationID      string          `json:"operation_id"`
	LeaseID          string          `json:"lease_id"`
	Epoch            int64           `json:"epoch"`
	State            string          `json:"state"`
	LocalState       string          `json:"local_state"`
	Payload          json.RawMessage `json:"payload,omitempty"`
	Artifacts        []Blob          `json:"artifacts,omitempty"`
	CleanupConfirmed bool            `json:"cleanup_confirmed"`
}
type Lease struct {
	Owner           string `json:"owner"`
	ID              string `json:"lease_id"`
	ControllerID    string `json:"controller_id"`
	HostID          string `json:"host_id"`
	HostInstanceID  string `json:"host_instance_id"`
	Epoch           int64  `json:"epoch"`
	State           string `json:"state"`
	LastKnownState  string `json:"last_known_state"`
	AndroidSlots    int    `json:"android_slots"`
	ManifestDigest  string `json:"manifest_digest"`
	PlanDigest      string `json:"plan_digest"`
	SourceSetDigest string `json:"source_set_digest"`
	RepositoryID    string `json:"repository_id"`
}
type Enrollment struct {
	Fingerprint string `json:"fingerprint"`
	Role        string `json:"role"`
	HostID      string `json:"host_id,omitempty"`
}
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
