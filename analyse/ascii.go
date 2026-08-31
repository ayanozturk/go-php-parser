package analyse

import (
	"strings"
	"unsafe"
)

// asciiLowerIdent lowercases PHP identifiers without the unicode.ToLower
// tables. Non-ASCII input falls back to strings.ToLower. Already-lowercase
// ASCII is returned unchanged and does not allocate.
func asciiLowerIdent(s string) string {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x80 {
			return strings.ToLower(s)
		}
		if c >= 'A' && c <= 'Z' {
			b := make([]byte, len(s))
			copy(b, s)
			b[i] = c + ('a' - 'A')
			for j := i + 1; j < len(s); j++ {
				cj := s[j]
				if cj >= 0x80 {
					return strings.ToLower(s)
				}
				if cj >= 'A' && cj <= 'Z' {
					b[j] = cj + ('a' - 'A')
				}
			}
			return bytesAsString(b)
		}
	}
	return s
}

func bytesAsString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
