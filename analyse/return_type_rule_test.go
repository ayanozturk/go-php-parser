package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/syntax"
)

// helper to run analysis on a PHP snippet and return issues
func analysePHP(t *testing.T, code string) []AnalysisIssue {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(code))
	if len(diags) > 0 {
		t.Fatalf("parser errors: %v", diags)
	}
	return RunAnalysisRules("test.php", nodes)
}

func hasReturnTypeIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == "A.RETURN.TYPE" {
			return true
		}
	}
	return false
}

func TestImplodeReturnsStringNoMismatch(t *testing.T) {
	php := `<?php
    function foo(): string {
        $arr = ["a", "b"];
        return implode("\n", $arr);
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for implode returning string, got: %#v", issues)
	}
}

func TestShortArrayLiteralReturnMatchesArrayType(t *testing.T) {
	php := `<?php
    function values(): array {
        return ['name'];
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for short array literal returned as array, got: %#v", issues)
	}
}

// TestReturnTypeRuleFallbackHonorsContentContext exercises
// collectReturnTypeIssues's ctx.Content branch specifically, not just the
// return-type rule in general. It deliberately passes an empty `nodes`
// slice (stale/mismatched relative to ctx.Content) and an AnalysisLevel
// below 2 so that neither ensureSharedFileDiagnostics(FromCST) nor
// ensureStructuralIssues(FromCST) pre-populates ctx.returnTypeIssues /
// ctx.hasReturnTypeIssues before collectReturnTypeIssues runs (both gate
// return-type collection behind analysisLevelAtLeast(ctx, 2)). That forces
// returnTypeIssuesForFile to fall through into collectReturnTypeIssues
// itself. Level < 2 also falls below appendReturnTypeOnNode's internal
// analysisLevelAtLeast(ctx, 3) gate, so a plain return-type mismatch
// wouldn't surface either way; the void-pure-function case
// (voidPureCode) is emitted unconditionally regardless of level, so it's
// used here as the observable signal.
//
// If collectReturnTypeIssues's ctx.Content branch were removed, this test
// would walk the empty `nodes` slice instead of re-parsing ctx.Content,
// find no functions, and fail to observe the issue - proving this test
// actually covers that branch rather than passing coincidentally.
func TestReturnTypeRuleFallbackHonorsContentContext(t *testing.T) {
	src := []byte(`<?php function f(): void { return; }`)
	rule := &ReturnTypeRule{}

	level := 0
	ctx := &AnalysisContext{Content: src, AnalysisLevel: &level}
	issues := rule.CheckIssues(nil, "test.php", ctx)

	found := false
	for _, iss := range issues {
		if iss.Code == voidPureCode {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected an %s issue produced from ctx.Content via collectReturnTypeIssues's fallback branch, got: %#v", voidPureCode, issues)
	}
}

func TestBinaryAndUnaryExpressionReturnTypes(t *testing.T) {
	tests := []struct {
		name       string
		returnType string
		expression string
		wantIssue  bool
	}{
		{name: "integer arithmetic", returnType: "int", expression: "1 + 2"},
		{name: "arithmetic mismatch", returnType: "string", expression: "1 + 2", wantIssue: true},
		{name: "comparison", returnType: "bool", expression: "1 < 2"},
		{name: "comparison mismatch", returnType: "string", expression: "1 < 2", wantIssue: true},
		{name: "logical", returnType: "bool", expression: "true && false"},
		{name: "logical mismatch", returnType: "string", expression: "true && false", wantIssue: true},
		{name: "unary not", returnType: "bool", expression: "!1"},
		{name: "unary numeric", returnType: "int", expression: "-1"},
		{name: "spaceship", returnType: "int", expression: "1 <=> 2"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			php := "<?php function value(): " + test.returnType + " { return " + test.expression + "; }"
			issues := analysePHP(t, php)
			if got := hasReturnTypeIssue(issues); got != test.wantIssue {
				t.Fatalf("has A.RETURN.TYPE = %t, want %t; issues: %#v", got, test.wantIssue, issues)
			}
		})
	}
}

func TestMultipleCompatibleTypesNoError(t *testing.T) {
	php := `<?php
    function bar(): bool {
        if ($x) {
            return true;
        }
        return $y; // mixed
    }`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue when actual types are [bool,mixed] declared bool, got: %#v", issues)
	}
}

func TestAssignedVariableReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	function foo(): string {
		$value = "ok";
		return $value;
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for assigned local string, got: %#v", issues)
	}
}

func TestNewExpressionClassReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class User {}

	function makeUser(): User {
		return new User();
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for new User return, got: %#v", issues)
	}
}

func TestThisPropertyReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class User {}

	class UserRepository {
		private User $user;

		public function current(): User {
			return $this->user;
		}
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for typed property fetch, got: %#v", issues)
	}
}

func TestInheritedGenericThisMethodReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
class User {}

/** @template T */
class Repository {
	/** @return T|null */
	public function find(mixed $id): ?object { return null; }
}

/** @extends Repository<User> */
class UserRepository extends Repository {
	public function byId(mixed $id): ?User {
		return $this->find($id);
	}
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		// ResolveMethod must bind @extends Repository<User> so find() returns User|null, not ?object.
		t.Fatalf("expected no A.RETURN.TYPE for inherited generic $this->find() bound to ?User, got: %#v", issues)
	}
}

func TestInheritedGenericThisMethodReturnTypeMismatchStillReports(t *testing.T) {
	php := `<?php
class User {}

/** @template T */
class Repository {
	/** @return T|null */
	public function find(mixed $id): ?object { return null; }
}

/** @extends Repository<User> */
class UserRepository extends Repository {
	public function byId(mixed $id): string {
		return $this->find($id);
	}
}`
	issues := analysePHP(t, php)
	if !hasReturnTypeIssue(issues) {
		t.Fatalf("expected A.RETURN.TYPE when returning bound generic ?User as string, got: %#v", issues)
	}
}

func TestCallableTemplateCacheReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
class Widget {}

interface Cache {
	/**
	 * @template T
	 * @param callable(): T $callback
	 * @return T
	 */
	public function get(string $key, callable $callback): mixed;
}

class Service {
	private Cache $cache;

	public function widget(): Widget {
		return $this->cache->get('k', function (): Widget {
			return new Widget();
		});
	}
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE for callable-bound cache get() returning Widget, got: %#v", issues)
	}
}

func TestCallableTemplateCacheReturnTypeMismatchStillReports(t *testing.T) {
	php := `<?php
class Widget {}

interface Cache {
	/**
	 * @template T
	 * @param callable(): T $callback
	 * @return T
	 */
	public function get(string $key, callable $callback): mixed;
}

class Service {
	private Cache $cache;

	public function widget(): string {
		return $this->cache->get('k', function (): Widget {
			return new Widget();
		});
	}
}`
	issues := analysePHP(t, php)
	if !hasReturnTypeIssue(issues) {
		t.Fatalf("expected A.RETURN.TYPE when cache get() returns Widget as string, got: %#v", issues)
	}
}

func TestGenericParentPropertyFindReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
class User {}

/** @template T */
class Repo {
	/** @return T|null */
	public function find($id): ?object { return null; }
}

/** @extends Repo<User> */
class UserRepo extends Repo {}

class Service {
	public UserRepo $repository;

	public function byId($id): ?User {
		return $this->repository->find($id);
	}
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE for @extends generic property find() bound to ?User, got: %#v", issues)
	}
}

func TestGenericParentDoctrineFindReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
class Record {}

/** @template T */
class EntityRepository {
	/** @return T|null */
	public function find($id): ?object { return null; }
}

class ServiceEntityRepository extends EntityRepository {}

/** @extends ServiceEntityRepository<Record> */
class RecordRepository extends ServiceEntityRepository {}

class Lookup {
	public RecordRepository $repository;
	public function byId($id): ?Record {
		return $this->repository->find($id);
	}
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE for multi-hop @extends ServiceEntityRepository find() bound to ?Record, got: %#v", issues)
	}
}

const doctrineVendorFindPHP = `<?php
namespace Doctrine\ORM;
/**
 * @template T of object
 */
class EntityRepository {
	/**
	 * @return object|null
	 * @phpstan-return ?T
	 */
	public function find(mixed $id): object|null { return null; }
}

namespace Doctrine\Bundle\DoctrineBundle\Repository;
use Doctrine\ORM\EntityRepository;
/**
 * @template T of object
 * @template-extends EntityRepository<T>
 */
class ServiceEntityRepository extends EntityRepository {}

namespace App\Entity;
class DocumentPolicy {}

namespace App\Repository;
use App\Entity\DocumentPolicy;
use Doctrine\Bundle\DoctrineBundle\Repository\ServiceEntityRepository;
/**
 * @extends ServiceEntityRepository<DocumentPolicy>
 */
class DocumentPolicyRepository extends ServiceEntityRepository {}

namespace App\Service;
use App\Entity\DocumentPolicy;
use App\Repository\DocumentPolicyRepository;
class DocumentPolicyService {
	public function __construct(private readonly DocumentPolicyRepository $repository) {}
	public function getPolicyById(string $id): ?DocumentPolicy {
		return $this->repository->find($id);
	}
}
`

func TestDoctrineVendorFindPhpstanReturnWithProjectIndex(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": doctrineVendorFindPHP}, 3)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE for namespaced Doctrine @phpstan-return ?T find() bound via @extends, got: %#v", issues)
	}
}

func TestAbstractRepositoryFindTemplateReturnWithProjectIndex(t *testing.T) {
	php := `<?php
interface EntityInterface {}
class Record implements EntityInterface {}

/** @template T of object */
class EntityRepository {
	/**
	 * @return object|null
	 * @phpstan-return T|null
	 */
	public function find($id): ?object { return null; }
}

class ServiceEntityRepository extends EntityRepository {}

/**
 * @template T of EntityInterface
 * @template-extends ServiceEntityRepository<T>
 */
abstract class AbstractRepository extends ServiceEntityRepository {
	/** @var EntityRepository<T>|null */
	private ?EntityRepository $resolvedRepository = null;

	/** @return T|null */
	public function find($id): ?object {
		return $this->resolveRepository()->find($id);
	}

	/** @return EntityRepository<T> */
	private function resolveRepository(): EntityRepository {
		if ($this->resolvedRepository instanceof EntityRepository) {
			return $this->resolvedRepository;
		}
		$this->resolvedRepository = new EntityRepository();
		return $this->resolvedRepository;
	}
}

/**
 * @extends AbstractRepository<Record>
 */
class RecordRepository extends AbstractRepository {}
`
	issues := runAnalysisLevelOnFiles(t, map[string]string{"test.php": php}, 3)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE for AbstractRepository find() wrapping EntityRepository<T>, got: %#v", issues)
	}
}

func TestThisMethodReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class User {}

	class UserRepository {
		public function current(): User {
			return $this->loadUser();
		}

		private function loadUser(): User {
			return new User();
		}
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for same-class method return, got: %#v", issues)
	}
}

func TestPromotedPropertyReturnTypeNoMismatch(t *testing.T) {
	php := `<?php
	class SessionStore {}

	class Session
	{
		public function __construct(private SessionStore $session)
		{
		}

		public function store(): SessionStore
		{
			return $this->session;
		}
	}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for promoted property fetch, got: %#v", issues)
	}
}

func TestLazyInitPropertyReturnTypeNoMismatch(t *testing.T) {
	// Lazy-init pattern: $this->prop is ?Type, assigned inside
	// `if (null === $this->prop)`, then returned as Type.
	php := `<?php
class MemberService {}
class Example {
    private ?MemberService $memberService = null;

    public function getMemberService(): MemberService
    {
        if (null === $this->memberService) {
            $this->memberService = new MemberService();
        }
        return $this->memberService;
    }
}`
	issues := analysePHP(t, php)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("expected no A.RETURN.TYPE issue for lazy-init property, got: %#v", issues)
	}
}

func TestReturnTypeRuleUsesSnapshotInferredTypeFact(t *testing.T) {
	const filename = "src/Answer.php"
	nodes, expression := parseReturnFactFixture(t, `<?php
function answer(): string {
    return 42;
}
`)
	rule := &ReturnTypeRule{}
	if issues := rule.CheckIssues(nodes, filename, &AnalysisContext{}); !hasReturnTypeIssue(issues) {
		t.Fatalf("expected ordinary inference to report int returned as string, got %#v", issues)
	}

	key := inferredTypeFactKey(filename, expression)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, []SemanticFact{{Key: key, Type: "string"}})
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	if issues := rule.CheckIssues(nodes, filename, snapshot.NewAnalysisContext()); hasReturnTypeIssue(issues) {
		t.Fatalf("expected shared inferred-type fact to satisfy declared return type, got %#v", issues)
	}
}

func TestReturnTypeRuleIgnoresMismatchedAndEmptyInferredTypeFacts(t *testing.T) {
	const filename = "src/Answer.php"
	nodes, expression := parseReturnFactFixture(t, `<?php
function answer(): string {
    return 42;
}
`)
	key := inferredTypeFactKey(filename, expression)
	mismatched := key
	mismatched.StartOffset++
	reader := &countingFactReader{facts: map[SemanticFactKey]SemanticFact{
		mismatched: {Key: mismatched, Type: "string"},
	}}

	issues := (&ReturnTypeRule{}).CheckIssues(nodes, filename, &AnalysisContext{Facts: reader})
	if !hasReturnTypeIssue(issues) {
		t.Fatalf("expected nonmatching fact span to fall back to ordinary inference, got %#v", issues)
	}
	if reader.lookups != 2 {
		t.Fatalf("expected two fact lookups (inferred-type, narrowed-type), got %d", reader.lookups)
	}

	reader.facts[key] = SemanticFact{Key: key}
	issues = (&ReturnTypeRule{}).CheckIssues(nodes, filename, &AnalysisContext{Facts: reader})
	if !hasReturnTypeIssue(issues) {
		t.Fatalf("expected empty inferred type fact to fall back to ordinary inference, got %#v", issues)
	}
}

type countingFactReader struct {
	facts   map[SemanticFactKey]SemanticFact
	lookups int
}

func (r *countingFactReader) Fact(key SemanticFactKey) (SemanticFact, bool) {
	r.lookups++
	fact, ok := r.facts[key]
	return fact, ok
}

func (r *countingFactReader) FactsForFile(string) []SemanticFact {
	return nil
}

func parseReturnFactFixture(t *testing.T, source string) ([]ast.Node, ast.Node) {
	t.Helper()
	nodes, diags := syntax.ParseAST([]byte(source))
	if len(diags) != 0 {
		t.Fatalf("parse errors: %v", diags)
	}
	fn, ok := nodes[0].(*ast.FunctionNode)
	if !ok || len(fn.Body) != 1 {
		t.Fatalf("expected one function with one statement, got %#v", nodes)
	}
	ret, ok := fn.Body[0].(*ast.ReturnNode)
	if !ok || ret.Expr == nil {
		t.Fatalf("expected return expression, got %#v", fn.Body[0])
	}
	return nodes, ret.Expr
}

func inferredTypeFactKey(filename string, expression ast.Node) SemanticFactKey {
	return SemanticFactKey{
		File:        filename,
		StartOffset: expression.GetPos().Offset,
		EndOffset:   expression.GetEndPos().Offset,
		Kind:        FactKindInferredType,
	}
}

func TestInferThisPropertyWithoutScopeIsConservative(t *testing.T) {
	fetch := &ast.PropertyFetchNode{
		Object:   &ast.VariableNode{Name: "this"},
		Property: "service",
	}
	if inferred := inferPropertyFetchType(fetch, nil, nil); inferred.String() != "mixed" {
		t.Fatalf("scope-less this-property type = %q, want mixed", inferred.String())
	}
}

func TestReturnTypePreservesNativeNullBesideArrayShapePHPDoc(t *testing.T) {
	issues := analysePHP(t, `<?php
/**
 * @return array{
 *     label: string
 * }|null
 */
function optional_payload(bool $available): ?array {
    if (!$available) {
        return null;
    }
    return ['label' => 'ready'];
}
`)
	if hasReturnTypeIssue(issues) {
		t.Fatalf("native nullable return was lost beside array-shape PHPDoc: %#v", issues)
	}
}

func TestReturnTypeKeepsClassTemplateIdentity(t *testing.T) {
	const filename = "src/Store.php"
	nodes := parseReturnCompletenessPHP(t, `<?php
namespace Example;

/** @template T of object */
abstract class Store {
    /** @return T|null */
    abstract protected function stored(): ?object;

    /** @return T|null */
    public function current(): ?object {
        return $this->stored();
    }
}
`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	issues := (&ReturnTypeRule{}).CheckIssues(nodes, filename, snapshot.NewAnalysisContext())
	if hasReturnTypeIssue(issues) {
		t.Fatalf("class template was resolved as a namespaced class: %#v", issues)
	}
}

func TestReturnTypeUsesLiteralFalseAndFopenSignature(t *testing.T) {
	const filename = "src/Streams.php"
	nodes := parseReturnCompletenessPHP(t, `<?php
/** @return resource|false */
function open_stream(string $path) {
    if ($path === '') {
        return false;
    }
    return fopen($path, 'rb');
}
`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build semantic snapshot: %v", err)
	}
	issues := (&ReturnTypeRule{}).CheckIssues(nodes, filename, snapshot.NewAnalysisContext())
	if hasReturnTypeIssue(issues) {
		t.Fatalf("resource-or-false returns should accept false and fopen: %#v", issues)
	}
}
