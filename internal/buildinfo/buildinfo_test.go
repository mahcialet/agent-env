package buildinfo

import "testing"

func TestCurrentHasHonestDevelopmentIdentity(t *testing.T) {
	got := Current()
	if got.Version != Version || got.GoVersion == "" || got.GOOS == "" || got.GOARCH == "" {
		t.Fatalf("unexpected build info: %+v", got)
	}
}
