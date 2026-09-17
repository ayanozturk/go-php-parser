package analyse

import (
	"testing"
)

func TestCheckClassModelIssuesFromCSTMatchesASTPath(t *testing.T) {
	cases := map[string]string{
		"finalAbstractConflict": `<?php
final abstract class Foo {}
`,
		"extendsUnknownClass": `<?php
class Foo extends Missing {}
`,
		"extendsWrongKind": `<?php
interface Bar {}
class Foo extends Bar {}
`,
		"extendsFinalParent": `<?php
final class Base {}
class Foo extends Base {}
`,
		"readonlyChildNonReadonlyParent": `<?php
class Base {}
readonly class Foo extends Base {}
`,
		"nonReadonlyChildReadonlyParent": `<?php
readonly class Base {}
class Foo extends Base {}
`,
		"implementsUnknownInterface": `<?php
class Foo implements Missing {}
`,
		"implementsWrongKind": `<?php
class Base {}
class Foo implements Base {}
`,
		"abstractMethodInNonAbstractClass": `<?php
class Foo {
    abstract public function bar();
}
`,
		"abstractMethodPrivate": `<?php
abstract class Foo {
    abstract private function bar();
}
`,
		"finalMethodOverride": `<?php
class Base {
    final public function bar() {}
}
class Foo extends Base {
    public function bar() {}
}
`,
		"finalPrivateConstantConflict": `<?php
class Foo {
    private final const X = 1;
}
`,
		"finalConstantOverride": `<?php
class Base {
    final const X = 1;
}
class Foo extends Base {
    const X = 2;
}
`,
		"interfaceExtendsUnknown": `<?php
interface Foo extends Missing {}
`,
		"interfaceExtendsWrongKind": `<?php
class Base {}
interface Foo extends Base {}
`,
		"traitUseUnknownTrait": `<?php
class Foo {
    use MissingTrait;
}
`,
		"traitUseWrongKind": `<?php
class Base {}
class Foo {
    use Base;
}
`,
		"traitUsingUnknownTrait": `<?php
trait Foo {
    use MissingTrait;
}
`,
		"anonymousClassExtendsUnknown": `<?php
function make() {
    return new class extends Missing {};
}
`,
		"backedEnumInvalidType": `<?php
enum Foo: float {
    case Bar;
}
`,
		"enumImplementsSerializable": `<?php
enum Foo implements Serializable {
    case Bar;
}
`,
		"readonlyPropertyOverrideMismatch": `<?php
class Base {
    public readonly int $x;
}
class Foo extends Base {
    public int $x;
}
`,
		"cleanClassHierarchy": `<?php
interface Greetable {
    public function greet(): string;
}
trait Helper {
    public function help(): void {}
}
class Base implements Greetable {
    use Helper;
    public function greet(): string { return "hi"; }
}
class Derived extends Base {
    public function greet(): string { return "hello"; }
}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, fileCtx, _, nodes := buildTypeRefTestContext(t, filename, src)
			want := sortIssuesForCompare((&Level0Rule{}).checkClassModel(filename, nodes, ctx, fileCtx))
			got := sortIssuesForCompare(CheckClassModelIssuesFromCST(filename, []byte(src), ctx))
			if len(want) != len(got) {
				t.Fatalf("issue count mismatch: ast=%d cst=%d\nast=%+v\ncst=%+v", len(want), len(got), want, got)
			}
			for i := range want {
				if want[i].Line != got[i].Line || want[i].Column != got[i].Column || want[i].Message != got[i].Message {
					t.Fatalf("issue %d mismatch:\nast=%+v\ncst=%+v", i, want[i], got[i])
				}
			}
		})
	}
}

func TestCheckClassModelIssuesFromCSTNilRoot(t *testing.T) {
	if got := CheckClassModelIssuesFromCST("empty.php", nil, &AnalysisContext{}); got != nil {
		t.Fatalf("expected nil issues for empty content, got %+v", got)
	}
}
