package domain

import "time"

// BrowserBinding selects a capability of an existing process, never its lifecycle.
type BrowserBinding struct {
	Name    string `json:"name"`
	Runtime string `json:"runtime"`
	CDPPort string `json:"cdp_port"`
}
type BrowserIdentity struct {
	Runtime   string `json:"runtime"`
	PID       int    `json:"pid"`
	Birth     string `json:"birth"`
	Port      int    `json:"port"`
	WebSocket string `json:"websocket"`
	Product   string `json:"product"`
	Protocol  string `json:"protocol"`
}
type BrowserPage struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
}
type BrowserNode struct {
	States      map[string]string `json:"states,omitempty"`
	Ref         string            `json:"ref"`
	BackendID   int               `json:"backend_id"`
	Frame       string            `json:"frame"`
	Role        string            `json:"role"`
	Name        string            `json:"name"`
	Value       string            `json:"value,omitempty"`
	Ignored     bool              `json:"ignored"`
	Disabled    bool              `json:"disabled"`
	Editable    bool              `json:"editable"`
	Password    bool              `json:"password"`
	Fingerprint string            `json:"fingerprint"`
}
type BrowserSnapshot struct {
	Version    int             `json:"version"`
	ID         string          `json:"id"`
	LeaseID    string          `json:"lease_id"`
	Browser    string          `json:"browser"`
	Identity   BrowserIdentity `json:"identity"`
	Page       BrowserPage     `json:"page"`
	Document   string          `json:"document"`
	CapturedAt time.Time       `json:"captured_at"`
	Nodes      []BrowserNode   `json:"nodes"`
	Truncated  bool            `json:"truncated"`
}

// Request fields are transient; input text and raw URL are not run argv.
type BrowserRequest struct {
	Operation string
	Page      string
	URL       string
	Text      string
	Key       string
	Contains  string
	Role      string
	WaitFor   string
	DeltaX    int
	DeltaY    int
	Duration  time.Duration
	Prior     *BrowserSnapshot
	Node      string
}
type BrowserObservation struct {
	Identity        BrowserIdentity  `json:"identity"`
	Pages           []BrowserPage    `json:"pages,omitempty"`
	Page            BrowserPage      `json:"page"`
	Snapshot        *BrowserSnapshot `json:"snapshot,omitempty"`
	DOM             []byte           `json:"-"`
	PNG             []byte           `json:"-"`
	Console         []BrowserConsole `json:"console,omitempty"`
	Network         []BrowserNetwork `json:"network,omitempty"`
	Truncated       bool             `json:"truncated"`
	Confirmed       bool             `json:"confirmed"`
	ActionPerformed bool             `json:"action_performed"`
	ReadbackEqual   bool             `json:"readback_equal"`
	Detail          string           `json:"detail,omitempty"`
}
type BrowserConsole struct {
	Type      string  `json:"type"`
	Text      string  `json:"text"`
	Timestamp float64 `json:"timestamp"`
}
type BrowserNetwork struct {
	ID        string  `json:"id"`
	URL       string  `json:"url"`
	Method    string  `json:"method"`
	Status    int     `json:"status"`
	Type      string  `json:"type"`
	Timestamp float64 `json:"timestamp"`
	Failure   string  `json:"failure,omitempty"`
}
