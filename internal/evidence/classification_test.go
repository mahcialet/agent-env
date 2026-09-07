package evidence

import "testing"

func TestBooleanTokenControlIsNotCredential(t *testing.T) {
	if got := Secrets(map[string]string{"HF_HUB_DISABLE_IMPLICIT_TOKEN": "1"}); len(got) != 0 {
		t.Fatal("public boolean flag classified as credential")
	}
	if got := Secrets(map[string]string{"API_TOKEN": "1"}); len(got) != 1 || got[0] != "1" {
		t.Fatal("actual short token must still be redacted")
	}
}
