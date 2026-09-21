package helper

import "testing"

func TestTrimWhitespaceAndClassDecl(t *testing.T) {
	if TrimWhitespace("  ab\t ") != "ab" {
		t.Fatal("trim")
	}
	if TrimWhitespace("") != "" {
		t.Fatal("empty trim")
	}
	for _, line := range []string{"class Foo", "interface I", "trait T", "enum E"} {
		if !IsClassDeclaration(line) {
			t.Fatalf("expected class-like: %q", line)
		}
	}
	for _, line := range []string{"", "// class X", "# class X", "classname", "  "} {
		if IsClassDeclaration(line) {
			t.Fatalf("not a declaration: %q", line)
		}
	}
}

func TestWordHelpers(t *testing.T) {
	if IndexOfWord("foo bar baz", "bar") != 4 {
		t.Fatal("IndexOfWord")
	}
	if IndexOfWord("foobar", "bar") != -1 {
		t.Fatal("IndexOfWord should require word boundaries")
	}
	if !ContainsWord("a function x", "function") || HasWord("functional", "function") {
		t.Fatal("ContainsWord/HasWord")
	}
	if !IsWordChar('a') || !IsWordChar('_') || IsWordChar('-') {
		t.Fatal("IsWordChar")
	}
}

func TestStringCase(t *testing.T) {
	if PascalCase("") != "" || PascalCase("foo_bar") != "FooBar" {
		t.Fatalf("PascalCase: %q %q", PascalCase(""), PascalCase("foo_bar"))
	}
	if CamelCase("") != "" || CamelCase("Foo_Bar") != "fooBar" {
		t.Fatalf("CamelCase: %q", CamelCase("Foo_Bar"))
	}
	if ToLower('A') != 'a' || ToLower('z') != 'z' {
		t.Fatal("ToLower")
	}
	if ToUpper('a') != 'A' || ToUpper('Z') != 'Z' {
		t.Fatal("ToUpper")
	}
}

func TestMethodNameSpanAfterFunction(t *testing.T) {
	start, end, ok := MethodNameSpanAfterFunction("  function foo()", 3, "foo")
	if !ok || start != 12 || end != 15 {
		t.Fatalf("got %d %d %v", start, end, ok)
	}
	start, end, ok = MethodNameSpanAfterFunction("function & bar()", 1, "bar")
	if !ok || start != 12 || end != 15 {
		t.Fatalf("by-ref: %d %d %v", start, end, ok)
	}
	if _, _, ok := MethodNameSpanAfterFunction("", 1, "x"); ok {
		t.Fatal("empty")
	}
	if _, _, ok := MethodNameSpanAfterFunction("nope", 1, "x"); ok {
		t.Fatal("missing function")
	}
	if _, _, ok := MethodNameSpanAfterFunction("function other()", 1, "name"); ok {
		t.Fatal("name mismatch")
	}
	// Scan for function when col does not already point at it.
	start, end, ok = MethodNameSpanAfterFunction("public function baz()", 1, "baz")
	if !ok || start != 17 || end != 20 {
		t.Fatalf("scan: %d %d %v", start, end, ok)
	}
}

func TestCommentAndQuoteHelpers(t *testing.T) {
	if !SkipLineComment("") || !SkipLineComment("// x") || !SkipLineComment("#x") || SkipLineComment("code") {
		t.Fatal("SkipLineComment")
	}
	cs := &CommentState{}
	j := HandleBlockComment("/* hi */", 0, cs)
	if !cs.InBlockComment && j != 2 {
		// entered at 0, returns 2; still in comment until */
	}
	cs = &CommentState{InBlockComment: true}
	j = HandleBlockComment("*/ next", 0, cs)
	if cs.InBlockComment || j != 2 {
		t.Fatalf("exit block: in=%v j=%d", cs.InBlockComment, j)
	}
	cs = &CommentState{}
	j = HandleHeredocStart("<<<EOT", 0, cs)
	if !cs.InHeredoc || cs.HeredocEnd != "EOT" || j != 6 {
		t.Fatalf("heredoc start: %#v j=%d", cs, j)
	}
	if !HandleHeredocEnd("EOT", cs) || cs.InHeredoc {
		t.Fatalf("heredoc end: %#v", cs)
	}
	if HandleHeredocEnd("x", &CommentState{}) {
		t.Fatal("not in heredoc")
	}
	qs := &QuoteState{}
	j = HandleQuotes(`'a'`, 0, qs)
	if !qs.InSingle || j != 1 {
		t.Fatalf("single quote enter: %#v j=%d", qs, j)
	}
	j = HandleQuotes(`'a'`, 2, qs)
	if qs.InSingle {
		t.Fatal("single quote exit")
	}
	qs = &QuoteState{}
	j = HandleQuotes(`"\""`, 0, qs)
	if !qs.InDouble {
		t.Fatal("double enter")
	}
	j = HandleQuotes(`"\""`, 1, qs)
	if j != 3 {
		t.Fatalf("escape skip: j=%d", j)
	}
}
