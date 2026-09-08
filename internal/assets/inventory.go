package assets

// AssetInfo describes immutable bytes bundled with this executable. Metadata is
// available without reading state or initializing a capability.
type AssetInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

// Inventory returns the production bundle inventory. No companion is embedded
// yet; test fixtures and externally supplied helper APKs are not bundled assets.
func Inventory() []AssetInfo {
	return []AssetInfo{}
}
