package sharedcache

import (
	"go-phpcs/lexer"
	"go-phpcs/token"
	"sync"
)

var tokenCache sync.Map // map[string][]token.Token

// BatchTokenizeFiles tokenizes all files in the given map (filename -> content) in parallel and caches the result.
func BatchTokenizeFiles(fileContents map[string][]byte) {
	var wg sync.WaitGroup
	for filename, content := range fileContents {
		wg.Add(1)
		go func(fn string, src []byte) {
			defer wg.Done()
			lex := lexer.New(string(src))
			tokens := make([]token.Token, 0, 256)
			for {
				tok := lex.NextToken()
				tokens = append(tokens, tok)
				if tok.Type == token.T_EOF {
					break
				}
			}
			tokenCache.Store(fn, tokens)
		}(filename, content)
	}
	wg.Wait()
}

// GetCachedTokens returns the cached tokens for a file, or nil if not present.
func GetCachedTokens(filename string) []token.Token {
	if val, ok := tokenCache.Load(filename); ok {
		return val.([]token.Token)
	}
	return nil
}

// ClearTokenCache removes all cached tokens (for test isolation or memory management).
func ClearTokenCache() {
	tokenCache = sync.Map{}
}
