package analyse

import "github.com/ayanozturk/go-php-parser/syntax"

// sharedParseResult returns the CST for this analysis file, parsing at most
// once per AnalysisContext. Prefer ctx.Parsed when already set; otherwise
// parse content (or ctx.Content) and store the result on ctx.
func sharedParseResult(ctx *AnalysisContext, content []byte) *syntax.ParseResult {
	if ctx != nil && ctx.Parsed != nil {
		return ctx.Parsed
	}
	src := content
	if len(src) == 0 && ctx != nil {
		src = ctx.Content
	}
	if len(src) == 0 {
		return nil
	}
	res := syntax.Parse(src)
	if ctx != nil {
		ctx.Parsed = res
		ctx.syntaxLower = nil
		ctx.hasSyntaxRootFt = false
		if len(ctx.Content) == 0 {
			ctx.Content = src
		}
	}
	return res
}
