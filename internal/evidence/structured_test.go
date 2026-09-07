package evidence

import "testing"

func TestContainsSecretStringDistinguishesSchemaAndData(t *testing.T) {
	type configuration struct {
		Roots []string `json:"roots"`
		Data  map[string]any
	}
	for _, tc := range []struct {
		name   string
		value  any
		secret string
		want   bool
	}{
		{"schema", configuration{Roots: []string{"api"}}, "root", false},
		{"slice value", configuration{Roots: []string{"root-service"}}, "root", true},
		{"dynamic key", configuration{Data: map[string]any{"root-service": nil}}, "root", true},
		{"nested value", configuration{Data: map[string]any{"safe": []any{"prefix-root"}}}, "root", true},
		{"escaped value", configuration{Roots: []string{"a\"b\\c"}}, "a\"b\\c", true},
		{"short credential", configuration{Roots: []string{"1"}}, "1", true},
		{"redaction marker credential", configuration{Roots: []string{"[REDACTED]"}}, "[REDACTED]", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ContainsSecretString(tc.value, []string{tc.secret}); got != tc.want {
				t.Fatalf("credential detection = %v, want %v", got, tc.want)
			}
		})
	}
}
