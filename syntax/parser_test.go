package syntax

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestParseNameKinds(t *testing.T) {
	cases := []struct {
		src  string
		kind Kind
		text string
	}{
		{"Foo", KindUnqualifiedName, "Foo"},
		{"Foo\\Bar", KindQualifiedName, "Foo\\Bar"},
		{"\\Foo\\Bar", KindFullyQualifiedName, "\\Foo\\Bar"},
		{"namespace\\Foo", KindRelativeName, "namespace\\Foo"},
	}
	for _, tc := range cases {
		p := NewParser([]byte(tc.src))
		g := p.parseName()
		f := &File{Source: []byte(tc.src), Green: g}
		BindRed(f)
		if f.Root.Kind() != tc.kind {
			t.Fatalf("%q: kind=%s want %s", tc.src, f.Root.Kind(), tc.kind)
		}
		if NameText(f.Root) != tc.text {
			t.Fatalf("%q: text=%q want %q", tc.src, NameText(f.Root), tc.text)
		}
	}
}

func TestParseTypes(t *testing.T) {
	cases := []struct {
		src  string
		kind Kind
	}{
		{"?string", KindNullableType},
		{"int|string", KindUnionType},
		{"Foo&Bar", KindIntersectionType},
		{"(Foo&Bar)|null", KindUnionType},
		{"array", KindPrimitiveType},
		{"callable", KindPrimitiveType},
		{"callable(int): string", KindCallableType},
		{"\\App\\Foo", KindNamedType},
	}
	for _, tc := range cases {
		p := NewParser([]byte(tc.src))
		g := p.parseType()
		f := &File{Source: []byte(tc.src), Green: g}
		BindRed(f)
		if f.Root.Kind() != tc.kind {
			t.Fatalf("%q: kind=%s want %s", tc.src, f.Root.Kind(), tc.kind)
		}
		if Print(f.Root) != tc.src {
			t.Fatalf("%q identity failed: got %q", tc.src, Print(f.Root))
		}
	}
}

func TestParseAttributes(t *testing.T) {
	src := "#[\\Foo\\Bar(x: 1), Baz]"
	p := NewParser([]byte(src))
	g := p.parseAttributeList()
	f := &File{Source: []byte(src), Green: g, Lines: token.NewLineTable([]byte(src))}
	BindRed(f)
	if f.Root.Kind() != KindAttributeList {
		t.Fatalf("kind=%s", f.Root.Kind())
	}
	groups := f.Root.ChildrenOfKind(KindAttributeGroup)
	if len(groups) != 1 {
		t.Fatalf("groups=%d", len(groups))
	}
	attrs := groups[0].ChildrenOfKind(KindAttribute)
	if len(attrs) != 2 {
		t.Fatalf("attrs=%d", len(attrs))
	}
	if Print(f.Root) != src {
		t.Fatalf("identity: got %q", Print(f.Root))
	}
}

func TestParseHeredocAndString(t *testing.T) {
	src := "<<<EOT\nhello $x\nEOT"
	p := NewParser([]byte(src))
	g := p.parseHeredoc()
	f := &File{Source: []byte(src), Green: g}
	BindRed(f)
	if f.Root.Kind() != KindHeredoc {
		t.Fatalf("kind=%s", f.Root.Kind())
	}
	if Print(f.Root) != src {
		t.Fatalf("identity got %q", Print(f.Root))
	}
}

func TestParseFileIdentity(t *testing.T) {
	src := "<?php\n#[Attr]\nfunction f(?string $a): int|false {\n  $x = \"hi $a\";\n}\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	var foundFn bool
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if n.Kind() == KindFunctionDecl {
			foundFn = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	if !foundFn {
		t.Fatal("expected KindFunctionDecl in structured green tree")
	}
}

func TestParseClassLikeIdentity(t *testing.T) {
	src := "<?php\nfinal class C extends B implements I1, I2 {\n  public int $x;\n  public function f(): void {}\n}\ninterface I {}\ntrait T { use U; }\nenum E: string { case A = 'a'; }\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("class-like identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindClassDecl:     false,
		KindInterfaceDecl: false,
		KindTraitDecl:     false,
		KindEnumDecl:      false,
		KindModifierList:  false,
		KindPropertyDecl:  false,
		KindMemberList:    false,
	}
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
}

func TestParseNamespaceUseAndAsymmetricVisibility(t *testing.T) {
	src := "<?php\nnamespace App;\nuse Foo\\Bar as Baz;\nclass C {\n  public(set) string $x;\n}\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindNamespaceDecl: false,
		KindUseDecl:       false,
		KindModifierList:  false,
	}
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
	ns, aliases := NamespaceAndAliases(res.File)
	if ns != "App" {
		t.Fatalf("namespace=%q", ns)
	}
	if aliases["baz"] != `Foo\Bar` {
		t.Fatalf("aliases=%v", aliases)
	}
}

func TestParseNamespaceBlockStructured(t *testing.T) {
	src := "<?php\nnamespace App {\nuse Foo\\Bar as Baz;\n;\nclass C {}\nfunction f() {}\n}\n"
	res := ParseForIndex([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindStatementList: false,
		KindUseDecl:       false,
		KindEmptyStmt:     false,
		KindClassDecl:     false,
		KindFunctionDecl:  false,
	}
	foundTokenList := false
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		if n.Kind() == KindTokenList {
			foundTokenList = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
	if !foundTokenList {
		t.Fatal("expected KindTokenList for function body under ParseForIndex")
	}
	ns, aliases := NamespaceAndAliases(res.File)
	if ns != "App" {
		t.Fatalf("namespace=%q", ns)
	}
	if aliases["baz"] != `Foo\Bar` {
		t.Fatalf("aliases=%v", aliases)
	}
}

func TestParseFunctionBodyFullyStructured(t *testing.T) {
	src := "<?php\nfunction f($a) {\nif ($a) { return $a; }\nforeach ([1] as $x) { echo $x; }\n$y = 1;\n}\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindFunctionDecl:   false,
		KindStatementList:  false,
		KindIfStmt:         false,
		KindForeachStmt:    false,
		KindReturnStmt:     false,
		KindEchoStmt:       false,
		KindExpressionStmt: false,
	}
	var funcHasStmtListBody bool
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		if n.Kind() == KindFunctionDecl {
			for _, c := range n.Children() {
				if c.Kind() == KindStatementList {
					funcHasStmtListBody = true
				}
			}
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
	if !funcHasStmtListBody {
		t.Fatal("full Parse should structure function body as KindStatementList")
	}
}

func TestParseGroupUseStructured(t *testing.T) {
	src := "<?php\nuse Foo\\Bar\\{A, B as C};\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	foundGroup := false
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if n.Kind() == KindUseGroup {
			foundGroup = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	if !foundGroup {
		t.Fatal("expected KindUseGroup")
	}
	_, aliases := NamespaceAndAliases(res.File)
	if aliases["a"] != `Foo\Bar\A` || aliases["c"] != `Foo\Bar\B` {
		t.Fatalf("aliases=%v", aliases)
	}
}

func TestParseTopLevelConstDeclareGlobalEchoReturn(t *testing.T) {
	src := "<?php\ndeclare(strict_types=1);\nconst FOO = 1;\nglobal $a;\nstatic $b = 2;\necho $a;\nreturn $b;\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindDeclareStmt:   false,
		KindConstDecl:     false,
		KindGlobalStmt:    false,
		KindStaticVarStmt: false,
		KindEchoStmt:      false,
		KindReturnStmt:    false,
	}
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
}

func TestParseTraitAdaptationStructured(t *testing.T) {
	src := "<?php\nclass C {\nuse A, B {\nA::foo insteadof B;\nB::bar as protected baz;\n}\n}\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindUseTraitClause:      false,
		KindTraitAdaptationList: false,
		KindTraitAdaptation:     false,
		KindModifierList:        false,
	}
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
}

func TestParseControlFlowStructured(t *testing.T) {
	src := `<?php
if ($a) { echo $a; } elseif ($b) { echo $b; } else { echo 0; }
if ($c): echo $c; elseif ($d): echo $d; else: echo 0; endif;
while ($i) { break; continue 2; }
while ($j): echo $j; endwhile;
do { $i--; } while ($i);
for ($i = 0; $i < 3; $i++) { echo $i; }
for ($i = 0; $i < 3; $i++): echo $i; endfor;
foreach ($xs as $k => $v) { unset($v); }
foreach ($xs as $v): echo $v; endforeach;
switch ($x) { case 1: echo 1; break; default: echo 0; }
switch ($y): case 2: echo 2; break; default: echo 0; endswitch;
try { throw $e; } catch (Exception $ex) { echo $ex; } finally { echo 1; }
match ($x) { 1, 2 => 'a', default => 'b' };
$z = 1;
`
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindIfStmt:        false,
		KindElseIfClause:  false,
		KindElseClause:    false,
		KindWhileStmt:     false,
		KindDoWhileStmt:   false,
		KindForStmt:       false,
		KindForeachStmt:   false,
		KindSwitchStmt:    false,
		KindCaseClause:    false,
		KindDefaultClause: false,
		KindMatchExpr:     false,
		KindMatchArm:      false,
		KindTryStmt:       false,
		KindCatchClause:   false,
		KindFinallyClause: false,
		KindBreakStmt:     false,
		KindContinueStmt:  false,
		KindThrowStmt:     false,
		KindUnsetStmt:     false,
		KindExpressionStmt: false,
	}
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
}

func TestParseControlFlowInsideNamespace(t *testing.T) {
	src := "<?php\nnamespace App {\nif ($a) { return $a; }\nforeach ($xs as $x) { echo $x; }\n}\n"
	res := Parse([]byte(src))
	got := Print(res.File.Root)
	if got != src {
		t.Fatalf("file identity\nwant %q\ngot  %q", src, got)
	}
	want := map[Kind]bool{
		KindIfStmt:      false,
		KindForeachStmt: false,
		KindReturnStmt:  false,
		KindEchoStmt:    false,
	}
	var walk func(*RedNode)
	walk = func(n *RedNode) {
		if n == nil {
			return
		}
		if _, ok := want[n.Kind()]; ok {
			want[n.Kind()] = true
		}
		for _, c := range n.Children() {
			walk(c)
		}
	}
	walk(res.File.Root)
	for k, found := range want {
		if !found {
			t.Fatalf("expected %s in green tree", k)
		}
	}
}


