package domain

import (
	"errors"
	"time"
)

var ErrUIInput = errors.New("invalid UI request")

// UI observations contain Android accessibility semantics, never Flutter widgets.
type UINode struct {
	Ref           string `json:"ref"`
	Parent        string `json:"parent"`
	WindowID      int    `json:"window_id"`
	Package       string `json:"package"`
	Class         string `json:"class"`
	ResourceID    string `json:"resource_id"`
	Text          string `json:"text"`
	Description   string `json:"description"`
	Hint          string `json:"hint"`
	Bounds        [4]int `json:"bounds"`
	Clickable     bool   `json:"clickable"`
	LongClickable bool   `json:"long_clickable"`
	Enabled       bool   `json:"enabled"`
	Focusable     bool   `json:"focusable"`
	Focused       bool   `json:"focused"`
	Editable      bool   `json:"editable"`
	Password      bool   `json:"password"`
	Scrollable    bool   `json:"scrollable"`
	Selected      bool   `json:"selected"`
	Checkable     bool   `json:"checkable"`
	Checked       bool   `json:"checked"`
	Visible       bool   `json:"visible"`
	Important     bool   `json:"important"`
	Actions       []int  `json:"actions"`
	Fingerprint   string `json:"fingerprint"`
}
type UIWindow struct {
	Key    string `json:"key,omitempty"`
	ID     int    `json:"id"`
	Type   int    `json:"type"`
	Active bool   `json:"active"`
}
type UITree struct {
	Windows   []UIWindow `json:"windows"`
	Nodes     []UINode   `json:"nodes"`
	Truncated bool       `json:"truncated"`
}
type UISnapshot struct {
	Version    int       `json:"version"`
	ID         string    `json:"id"`
	LeaseID    string    `json:"lease_id"`
	Runtime    string    `json:"runtime"`
	Serial     string    `json:"serial"`
	Package    string    `json:"package"`
	Backend    string    `json:"backend"`
	CapturedAt time.Time `json:"captured_at"`
	Tree       UITree    `json:"tree"`
}

// UIRequest is an ephemeral device request. Text must never be persisted.
type UIRequest struct {
	Version             int    `json:"version"`
	Operation           string `json:"operation"`
	Package             string `json:"package,omitempty"`
	ExpectedFingerprint string `json:"expected_fingerprint,omitempty"`
	ExpectedBackend     string `json:"expected_backend,omitempty"`
	Text                string `json:"text,omitempty"`
	X                   int    `json:"x,omitempty"`
	Y                   int    `json:"y,omitempty"`
	ToX                 int    `json:"to_x,omitempty"`
	ToY                 int    `json:"to_y,omitempty"`
	DurationMS          int    `json:"duration_ms,omitempty"`
	SinceSeconds        int    `json:"since_seconds,omitempty"`
}
type UIObservation struct {
	Log              string `json:"log,omitempty"`
	Backend          string `json:"backend,omitempty"`
	Version          int    `json:"version"`
	Status           string `json:"status"`
	Snapshot         UITree `json:"snapshot"`
	ActionPerformed  bool   `json:"action_performed"`
	ReadbackEqual    bool   `json:"readback_equal"`
	AfterFingerprint string `json:"after_fingerprint,omitempty"`
	Detail           string `json:"detail,omitempty"`
	// Binary and Raw are already bounded at the adapter boundary.
	Binary    []byte `json:"-"`
	Raw       []byte `json:"-"`
	Truncated bool   `json:"truncated,omitempty"`
	PID       int    `json:"pid,omitempty"`
	Since     string `json:"since,omitempty"`
	// Confirmed means remote completion is known, including refusal.
	Confirmed       bool `json:"-"`
	HelperInstalled bool `json:"helper_installed,omitempty"`
}
