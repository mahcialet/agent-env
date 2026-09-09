package protocol

import (
	"errors"
	"strings"
	"testing"
)

func TestDurablePayloadRejectsTransientText(t *testing.T) {
	for _, kind := range []string{"ui", "browser"} {
		for _, payload := range []string{
			`{"ui":{"Operation":"set-text","Text":"review-secret"}}`,
			`{"browser":{"Request":{"operation":"set-text","text":""}}}`,
			`{"ui":{"Operation":"tap","Text":"review-secret"}}`,
			`{"ui":{"tExT":"review-secret","Text":""}}`,
			`{"ui":{"Text":"review-secret"},"ui":{}}`,
			`{"browser":{"Request":{"text":123}}}`,
		} {
			err := ValidateDurablePayload(kind, []byte(payload))
			if !errors.Is(err, ErrTransientInput) || strings.Contains(err.Error(), "review-secret") {
				t.Fatalf("%s transient input was not safely refused: %v", kind, err)
			}
		}
		for _, payload := range []string{`{"ui":{"Operation":"tap","Text":""}}`, `{"browser":{"Request":{"operation":"snapshot"}}}`} {
			if err := ValidateDurablePayload(kind, []byte(payload)); err != nil {
				t.Fatal(err)
			}
		}
	}
}
