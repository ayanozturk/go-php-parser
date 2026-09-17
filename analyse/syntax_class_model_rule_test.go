package analyse

import (
	"testing"
)

func TestCheckClassModelIssuesFromCST(t *testing.T) {
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

	type wantIssue struct {
		Message string
		Line    int
		Column  int
	}

	want := map[string][]wantIssue{
		"finalAbstractConflict": {
			{Message: "Class Foo cannot be both final and abstract.", Line: 2, Column: 1},
		},
		"extendsUnknownClass": {
			{Message: "Class Foo extends unknown class Missing.", Line: 2, Column: 1},
		},
		"extendsWrongKind": {
			{Message: "Class Foo extends interface Bar.", Line: 3, Column: 1},
		},
		"extendsFinalParent": {
			{Message: "Class Foo extends final class Base.", Line: 3, Column: 1},
		},
		"readonlyChildNonReadonlyParent": {
			{Message: "Readonly class Foo cannot extend non-readonly class Base.", Line: 3, Column: 1},
		},
		"nonReadonlyChildReadonlyParent": {
			{Message: "Non-readonly class Foo cannot extend readonly class Base.", Line: 3, Column: 1},
		},
		"implementsUnknownInterface": {
			{Message: "Class Foo implements unknown interface Missing.", Line: 2, Column: 1},
		},
		"implementsWrongKind": {
			{Message: "Class Foo implements class Base.", Line: 3, Column: 1},
		},
		"abstractMethodInNonAbstractClass": {
			{Message: "Class Foo has abstract method bar() but is not abstract.", Line: 3, Column: 21},
		},
		"abstractMethodPrivate": {
			{Message: "Abstract method Foo::bar() cannot be private.", Line: 3, Column: 22},
		},
		"finalMethodOverride": {
			{Message: "Cannot override final method Base::bar().", Line: 6, Column: 12},
		},
		"finalPrivateConstantConflict": {
			{Message: "Private constant Foo::X cannot be final.", Line: 3, Column: 25},
		},
		"finalConstantOverride": {
			{Message: "Cannot override final constant Base::X.", Line: 6, Column: 11},
		},
		"interfaceExtendsUnknown": {
			{Message: "Interface Foo extends unknown interface Missing.", Line: 2, Column: 1},
		},
		"interfaceExtendsWrongKind": {
			{Message: "Interface Foo extends class Base.", Line: 3, Column: 1},
		},
		"traitUseUnknownTrait": {
			{Message: "Trait MissingTrait not found.", Line: 3, Column: 5},
		},
		"traitUseWrongKind": {
			{Message: "Class Base used as trait.", Line: 4, Column: 5},
		},
		"traitUsingUnknownTrait": {
			{Message: "Trait MissingTrait not found.", Line: 3, Column: 5},
		},
		"anonymousClassExtendsUnknown": {
			{Message: "Class  extends unknown class Missing.", Line: 3, Column: 16},
		},
		"backedEnumInvalidType": {
			{Message: `Backed enum Foo can have only "int" or "string" type.`, Line: 2, Column: 1},
			{Message: `Enum case Foo::Bar does not have a value but the enum is backed with the "float" type.`, Line: 3, Column: 5},
		},
		"enumImplementsSerializable": {
			{Message: "Enum Foo cannot implement Serializable.", Line: 2, Column: 1},
		},
		"readonlyPropertyOverrideMismatch": {
			{Message: "Property Foo::$x overriding readonly property must be readonly.", Line: 6, Column: 5},
		},
		"cleanClassHierarchy": {},
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			filename := name + ".php"
			ctx, _, _, _ := buildTypeRefTestContext(t, filename, src)
			got := sortIssuesForCompare(CheckClassModelIssuesFromCST(filename, []byte(src), ctx))

			wantIssues := want[name]
			if len(wantIssues) != len(got) {
				t.Fatalf("issue count mismatch: want=%d got=%d\nwant=%+v\ngot=%+v", len(wantIssues), len(got), wantIssues, got)
			}
			for i := range wantIssues {
				if wantIssues[i].Line != got[i].Line || wantIssues[i].Column != got[i].Column || wantIssues[i].Message != got[i].Message {
					t.Fatalf("issue %d mismatch:\nwant=%+v\ngot=%+v", i, wantIssues[i], got[i])
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
