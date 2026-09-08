package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
)

// SecretFingerprint retains bounded matching evidence, never the entered text.
type SecretFingerprint struct {
	Length       int    `json:"length"`
	SHA256       string `json:"sha256"`
	PrefixSHA256 string `json:"prefix_sha256"`
}

func FingerprintSecret(s string) SecretFingerprint {
	n := len(s)
	prefix := n
	if prefix > 8 {
		prefix = 8
	}
	h := sha256.Sum256([]byte(s))
	p := sha256.Sum256([]byte(s[:prefix]))
	return SecretFingerprint{Length: n, SHA256: hex.EncodeToString(h[:]), PrefixSHA256: hex.EncodeToString(p[:])}
}

// RedactFingerprints fails closed on invalid evidence or excessive work.
func RedactFingerprints(s string, fs []SecretFingerprint) (string, error) {
	fail := func() (string, error) {
		return "", errors.New("browser redaction evidence is invalid or exceeds verification limits")
	}
	if len(s) > 16<<20 || len(fs) > 256 {
		return fail()
	}
	type entry struct {
		length int
		sum    [32]byte
	}
	indexes := map[int]map[[32]byte][]entry{}
	for _, f := range fs {
		if f.Length < 1 || f.Length > 4096 {
			return fail()
		}
		full, e := hex.DecodeString(f.SHA256)
		if e != nil || len(full) != 32 {
			return fail()
		}
		prefix, e := hex.DecodeString(f.PrefixSHA256)
		if e != nil || len(prefix) != 32 {
			return fail()
		}
		n := f.Length
		if n > 8 {
			n = 8
		}
		if indexes[n] == nil {
			indexes[n] = map[[32]byte][]entry{}
		}
		var p, h [32]byte
		copy(p[:], prefix)
		copy(h[:], full)
		indexes[n][p] = append(indexes[n][p], entry{length: f.Length, sum: h})
	}
	lengths := []int{}
	for n := range indexes {
		lengths = append(lengths, n)
	}
	sort.Ints(lengths)
	type span struct{ start, end int }
	matches := []span{}
	work := 0
	verified := 0
	for i := 0; i < len(s); i++ {
		for _, n := range lengths {
			if i+n > len(s) {
				continue
			}
			work++
			if work > 32<<20 {
				return fail()
			}
			h := sha256.Sum256([]byte(s[i : i+n]))
			for _, f := range indexes[n][h] {
				if i+f.length > len(s) {
					continue
				}
				verified += f.length
				if verified > 32<<20 {
					return fail()
				}
				if sha256.Sum256([]byte(s[i:i+f.length])) == f.sum {
					matches = append(matches, span{i, i + f.length})
					if len(matches) > 1<<20 {
						return fail()
					}
				}
			}
		}
	}
	if len(matches) == 0 {
		return s, nil
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return matches[i].end > matches[j].end
	})
	var out strings.Builder
	at := 0
	for i := 0; i < len(matches); {
		a, b := matches[i].start, matches[i].end
		i++
		for i < len(matches) && matches[i].start <= b {
			if matches[i].end > b {
				b = matches[i].end
			}
			i++
		}
		out.WriteString(s[at:a])
		out.WriteString("[REDACTED]")
		at = b
	}
	out.WriteString(s[at:])
	return out.String(), nil
}
