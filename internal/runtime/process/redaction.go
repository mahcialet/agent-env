package process

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

type redactionEvidence struct {
	Owner   ownership
	Version int
	Secrets []secretFingerprint
}

// Only irreversible digests are durable. Prefix digests avoid hashing every
// possible full-length substring, while the complete digest decides matches.
type secretFingerprint struct {
	Length               int
	Digest, PrefixDigest string
}

func fingerprintSecrets(values []string) ([]secretFingerprint, error) {
	var result []secretFingerprint
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		if len(value) > 1024*1024 || len(result) >= 4096 {
			return nil, errors.New("process secret evidence exceeds bounded limits")
		}
		full := sha256.Sum256([]byte(value))
		prefix := sha256.Sum256([]byte(value[:min(8, len(value))]))
		result = append(result, secretFingerprint{len(value), hex.EncodeToString(full[:]), hex.EncodeToString(prefix[:])})
	}
	return result, nil
}

type digestCandidate struct {
	length int
	digest [32]byte
}
type fingerprintRedactor struct {
	longest int
	buckets map[int]map[[32]byte][]digestCandidate
}

func newFingerprintRedactor(values []secretFingerprint) (fingerprintRedactor, error) {
	r := fingerprintRedactor{buckets: map[int]map[[32]byte][]digestCandidate{}}
	if len(values) > 4096 {
		return r, errors.New("oversized secret redaction evidence")
	}
	for _, value := range values {
		digest, e1 := hex.DecodeString(value.Digest)
		prefix, e2 := hex.DecodeString(value.PrefixDigest)
		if value.Length <= 0 || value.Length > 1024*1024 || e1 != nil || e2 != nil || len(digest) != 32 || len(prefix) != 32 {
			return r, errors.New("invalid secret redaction evidence")
		}
		prefixLength := min(8, value.Length)
		if r.buckets[prefixLength] == nil {
			r.buckets[prefixLength] = map[[32]byte][]digestCandidate{}
		}
		key := [32]byte(prefix)
		r.buckets[prefixLength][key] = append(r.buckets[prefixLength][key], digestCandidate{value.Length, [32]byte(digest)})
		r.longest = max(r.longest, value.Length)
	}
	return r, nil
}
func (r fingerprintRedactor) redact(ctx context.Context, data []byte, limit int) (string, error) {
	var output strings.Builder
	end := min(len(data), limit)
	checkedBytes, checkedCandidates := 0, 0
	redactedUntil := 0
	truncated := len(data) > limit
	for i := 0; i < end; {
		if i%4096 == 0 {
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}
		matched := 0
		for prefixLength, bucket := range r.buckets {
			if len(data)-i < prefixLength {
				continue
			}
			prefix := sha256.Sum256(data[i : i+prefixLength])
			for _, candidate := range bucket[prefix] {
				checkedCandidates++
				if checkedCandidates > 1024*1024 {
					return "", errors.New("log redaction candidate budget exceeded")
				}
				if candidate.length > len(data)-i {
					continue
				}
				checkedBytes += candidate.length
				if checkedBytes > 64*1024*1024 {
					return "", errors.New("log redaction verification budget exceeded")
				}
				if sha256.Sum256(data[i:i+candidate.length]) == candidate.digest {
					matched = max(matched, candidate.length)
				}
			}
		}
		if matched > 0 {
			if i >= redactedUntil {
				if output.Len()+len("[REDACTED]") > limit {
					truncated = true
					break
				}
				output.WriteString("[REDACTED]")
			}
			redactedUntil = max(redactedUntil, i+matched)
		}
		if i >= redactedUntil {
			if output.Len() >= limit {
				truncated = true
				break
			}
			output.WriteByte(data[i])
		}
		i++
	}
	if truncated {
		output.WriteString("\n[log truncated]\n")
	}
	return output.String(), nil
}
