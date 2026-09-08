package evidence

import (
	"strings"
	"testing"
)

func TestFingerprintRedactionUnicodeAndOverlap(t *testing.T) {
	fs := []SecretFingerprint{FingerprintSecret("秘密🙂"), FingerprintSecret("abc"), FingerprintSecret("bcd")}
	out, e := RedactFingerprints("later 秘密🙂 abcd again 秘密🙂", fs)
	if e != nil || out != "later [REDACTED] [REDACTED] again [REDACTED]" {
		t.Fatalf("%q %v", out, e)
	}
}
func TestFingerprintRedactionRejectsMalformedAndBounds(t *testing.T) {
	for _, fs := range [][]SecretFingerprint{{{Length: 4, SHA256: "invalid", PrefixSHA256: "invalid"}}, {FingerprintSecret("")}, {FingerprintSecret(strings.Repeat("x", 4097))}, make([]SecretFingerprint, 257)} {
		if out, e := RedactFingerprints("text", fs); e == nil || out != "" {
			t.Fatal("invalid evidence accepted")
		}
	}
	if out, e := RedactFingerprints(strings.Repeat("x", (16<<20)+1), nil); e == nil || out != "" {
		t.Fatal("unbounded content accepted")
	}
}
func TestFingerprintRedactionCandidateBudget(t *testing.T) {
	f := FingerprintSecret(strings.Repeat("a", 4096))
	if out, e := RedactFingerprints(strings.Repeat("a", 16384), []SecretFingerprint{f}); e == nil || out != "" {
		t.Fatal("verification budget ignored")
	}
}
func TestFingerprintRedactionPrefixCollisionDoesNotRedact(t *testing.T) {
	out, e := RedactFingerprints("sameprefother", []SecretFingerprint{FingerprintSecret("sameprefsecret")})
	if e != nil || out != "sameprefother" {
		t.Fatalf("%q %v", out, e)
	}
}
