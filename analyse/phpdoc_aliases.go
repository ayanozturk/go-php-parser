package analyse

import (
	"strings"
	"unicode/utf8"
)

func substitutePHPDocTypeAliasTokens(raw string, aliases map[string]string) string {
	if raw == "" || len(aliases) == 0 {
		return raw
	}
	var out strings.Builder
	quote := rune(0)
	for start := 0; start < len(raw); {
		current, size := utf8.DecodeRuneInString(raw[start:])
		if quote != 0 {
			out.WriteString(raw[start : start+size])
			if current == '\\' && start+size < len(raw) {
				_, nextSize := utf8.DecodeRuneInString(raw[start+size:])
				out.WriteString(raw[start+size : start+size+nextSize])
				start += size + nextSize
				continue
			}
			if current == quote {
				quote = 0
			}
			start += size
			continue
		}
		if current == '\'' || current == '"' {
			quote = current
			out.WriteString(raw[start : start+size])
			start += size
			continue
		}
		if !isTemplateIdentifierRune(current) {
			out.WriteString(raw[start : start+size])
			start += size
			continue
		}
		end := start + size
		for end < len(raw) {
			next, nextSize := utf8.DecodeRuneInString(raw[end:])
			if !isTemplateIdentifierRune(next) {
				break
			}
			end += nextSize
		}
		token := raw[start:end]
		if replacement, ok := aliases[token]; ok {
			out.WriteString(replacement)
		} else {
			out.WriteString(token)
		}
		start = end
	}
	return out.String()
}
