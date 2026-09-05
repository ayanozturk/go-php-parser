package analyse

import (
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
	"strings"
	"testing"
	"time"
)

func parsePHPForLevel0(t *testing.T, php string) []ast.Node {
	t.Helper()
	p := parser.New(lexer.New(php), false)
	nodes := p.Parse()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse errors: %v", p.Errors())
	}
	return nodes
}

func runLevel0OnFiles(t *testing.T, files map[string]string) []AnalysisIssue {
	return runAnalysisLevelOnFiles(t, files, 0)
}

func runLevel1OnFiles(t *testing.T, files map[string]string) []AnalysisIssue {
	return runAnalysisLevelOnFiles(t, files, 1)
}

func runAnalysisLevelOnFiles(t *testing.T, files map[string]string, level int) []AnalysisIssue {
	t.Helper()
	parsed := make(map[string][]ast.Node, len(files))
	for filename, php := range files {
		parsed[filename] = parsePHPForLevel0(t, php)
	}
	project := BuildProjectIndex(parsed)
	var issues []AnalysisIssue
	for filename, nodes := range parsed {
		ctx := &AnalysisContext{Resolver: project, AnalysisLevel: &level}
		issues = append(issues, RunAnalysisRulesWithContext(filename, nodes, ctx)...)
	}
	return issues
}

func TestLevel0DNFTypeReferencesResolveIndividualMembers(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
interface FirstContract {}
interface SecondContract {}
class Alternative {}
function run((FirstContract&SecondContract)|Alternative $value): void {}
`,
	}, 0)
	if hasIssueContaining(issues, level0SymbolsCode, "unknown class") {
		t.Fatalf("expected DNF members to resolve individually, got %#v", issues)
	}
}

func hasIssueContaining(issues []AnalysisIssue, code, needle string) bool {
	for _, issue := range issues {
		if issue.Code == code && strings.Contains(issue.Message, needle) {
			return true
		}
	}
	return false
}

func countIssueContaining(issues []AnalysisIssue, code, needle string) int {
	count := 0
	for _, issue := range issues {
		if issue.Code == code && strings.Contains(issue.Message, needle) {
			count++
		}
	}
	return count
}

func TestLevel0UnknownSymbolsAndFunctionArguments(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
known(tooMany: 1, 2);
missing_function();
new MissingClass();

function known($a) {}
`,
	})

	if !hasIssueContaining(issues, level0SymbolsCode, "Function missing_function not found") {
		t.Fatalf("expected unknown function issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Instantiated class MissingClass not found") {
		t.Fatalf("expected unknown class issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0InvocationCode, "Named argument cannot be followed by a positional argument") {
		t.Fatalf("expected named argument order issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0InvocationCode, "Unknown parameter $tooMany") {
		t.Fatalf("expected unknown named parameter issue, got %#v", issues)
	}
}

func TestLevel0RecognizesCommonArrayBuiltins(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
use function count;

function run(array $rows): array
{
    if (array_key_exists('name', $rows)) {
        return array_values(array_filter(array_column($rows, 'name')));
    }
    return [trim((string) count($rows))];
}
`,
	})

	for _, name := range []string{"array_column", "array_filter", "array_key_exists", "array_values", "count", "trim"} {
		if hasIssueContaining(issues, level0SymbolsCode, "Function "+name+" not found") {
			t.Fatalf("expected %s to be recognized as builtin, got %#v", name, issues)
		}
	}
}

func TestLevel0RecognizesEnumCasesAndNativeMethods(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
enum Status: string {
    case ACTIVE = 'active';
}

function run(): void
{
    Status::ACTIVE;
    Status::from('active');
    Status::tryFrom('missing');
    Status::cases();
}
`,
	})

	for _, unexpected := range []string{
		"Access to undefined constant Status::ACTIVE",
		"Call to an undefined static method Status::from",
		"Call to an undefined static method Status::tryFrom",
		"Call to an undefined static method Status::cases",
	} {
		if hasIssueContaining(issues, level0SymbolsCode, unexpected) {
			t.Fatalf("expected enum member %q to be recognized, got %#v", unexpected, issues)
		}
	}
}

func TestLevel0RecognizesEnumNativeProperties(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
enum InvoiceStatus: int {
    case PENDING = 1;
    case PAID = 2;

    public function isPending(): bool
    {
        return $this->value === self::PENDING->value;
    }

    public function label(): string
    {
        return $this->name;
    }
}

enum UnitStatus {
    case Ready;

    public function describe(): string
    {
        return $this->name . $this->value;
    }
}
`,
	})

	for _, unexpected := range []string{
		"Access to an undefined property InvoiceStatus::$value",
		"Access to an undefined property InvoiceStatus::$name",
		"Access to an undefined property UnitStatus::$name",
	} {
		if hasIssueContaining(issues, level0SymbolsCode, unexpected) {
			t.Fatalf("expected native enum property %q to be recognized, got %#v", unexpected, issues)
		}
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Access to an undefined property UnitStatus::$value") {
		t.Fatalf("expected unit enum $value to remain undefined, got %#v", issues)
	}
}

func TestLevel0BuiltinMethodsHaveUsableInvocationBounds(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
enum Status: string {
    case ACTIVE = 'active';
}

final class Demo {
    public function run(): void {
        Status::from('active');
        DateTime::createFromFormat('Y-m-d', '2026-05-27');
        new ReflectionClass(self::class);
    }
}
`,
	})

	for _, unexpected := range []string{
		"DateTime::createFromFormat() invoked with 2 parameters, at most 0 allowed",
		"ReflectionClass constructor invoked with 1 parameter, at most 0 allowed",
		"Status::from() invoked with 1 parameter, at most 0 allowed",
	} {
		if hasIssueContaining(issues, level0InvocationCode, unexpected) {
			t.Fatalf("unexpected invocation false positive %q, got %#v", unexpected, issues)
		}
	}
}

func TestLevel0RecognizesCoreDateTimeExceptionAndFilesystemBuiltins(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class AppError extends RuntimeException {}

function run(DateTimeInterface $clock, Throwable $error): string
{
    $now = new DateTime('now');
    $immutable = new DateTimeImmutable('tomorrow', new DateTimeZone('UTC'));
    $now->modify('+1 day');
    $now->setTime(1, 2, 3);
    json_encode(['ok' => true]);
    file_put_contents(tempnam(sys_get_temp_dir(), 'hr'), json_decode('{}'));
    throw new InvalidArgumentException('missing');
    return $clock->format('c') . $error->getMessage() . (new ReflectionClass(self::class))->getProperty('run')->getName();
}

function wrap(): AppError
{
    return new AppError('failed');
}
`,
	})

	for _, unexpected := range []string{
		"Instantiated class RuntimeException not found",
		"Instantiated class InvalidArgumentException not found",
		"Function json_encode not found",
		"Function json_decode not found",
		"Function file_put_contents not found",
		"Function tempnam not found",
		"Class DateTime constructor invoked with 1 parameter, at most 0 allowed",
		"Class DateTimeImmutable constructor invoked with 2 parameters, at most 0 allowed",
		"Class AppError constructor invoked with 1 parameter, at most 0 allowed",
		"extends unknown class RuntimeException",
	} {
		if hasIssueContaining(issues, level0SymbolsCode, unexpected) || hasIssueContaining(issues, level0InvocationCode, unexpected) || hasIssueContaining(issues, level0ClassModelCode, unexpected) {
			t.Fatalf("unexpected builtin false positive %q, got %#v", unexpected, issues)
		}
	}
}

func TestLevel2RecognizesDateTimeAndReflectionMethods(t *testing.T) {
	issues := runAnalysisLevelOnFiles(t, map[string]string{
		"test.php": `<?php
function run(DateTime $now, ReflectionClass $reflector, ReflectionProperty $property, ReflectionMethod $method): void
{
    $now->format('c');
    $now->modify('+1 day');
    $reflector->getProperty('name');
    $reflector->hasMethod('run');
    $property->setValue(null, 'x');
    $method->invoke(null);
}
`,
	}, 2)

	for _, unexpected := range []string{
		"Call to an undefined method DateTime::format()",
		"Call to an undefined method DateTime::modify()",
		"Call to an undefined method ReflectionClass::getProperty()",
		"Call to an undefined method ReflectionClass::hasMethod()",
		"Call to an undefined method ReflectionProperty::setValue()",
		"Call to an undefined method ReflectionMethod::invoke()",
	} {
		if hasIssueContaining(issues, level2MethodExistenceCode, unexpected) {
			t.Fatalf("unexpected method false positive %q, got %#v", unexpected, issues)
		}
	}
}

func TestLevel0ParentConstructorCallIsNotStaticInstanceCall(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Base {
    public function __construct(string $name) {}
}

class Child extends Base {
    public function __construct() {
        parent::__construct('child');
    }
}
`,
	})

	if hasIssueContaining(issues, level0InvocationCode, "Static call to instance method Base::__construct") {
		t.Fatalf("parent constructor call should not be reported as static instance call, got %#v", issues)
	}
}

func TestLevel0TraitPrivateAndProtectedMethodsAreCallableFromUsingClass(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
trait Helpers {
    private function privateHelper(): void {}
    protected function protectedHelper(): void {}
}

class UsesHelpers {
    use Helpers;

    public function run(): void {
        $this->privateHelper();
        $this->protectedHelper();
    }
}
`,
	})

	if hasIssueContaining(issues, level0InvocationCode, "Call to private method Helpers::privateHelper") {
		t.Fatalf("private trait method should be callable from using class, got %#v", issues)
	}
	if hasIssueContaining(issues, level0InvocationCode, "Call to protected method Helpers::protectedHelper") {
		t.Fatalf("protected trait method should be callable from using class, got %#v", issues)
	}
}

func TestLevel0DoesNotValidateAttributeConstructorArity(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
#[Attribute]
class NeedsArg {
    public function __construct(string $name) {}
}

#[NeedsArg]
class Demo {}
`,
	})

	if hasIssueContaining(issues, level0InvocationCode, "Attribute class NeedsArg constructor invoked with 0 parameters") {
		t.Fatalf("attribute constructor arity should not be reported at level 0, got %#v", issues)
	}
}

func TestLevel0TreatsClassImportsAsPotentialNamespaceAliases(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
namespace App;

use Doctrine\ORM\Mapping as ORM;

#[ORM\Entity]
class User {}
`,
	})

	if hasIssueContaining(issues, level0SymbolsCode, "Used class Doctrine\\ORM\\Mapping not found") {
		t.Fatalf("namespace-style class import should be checked at use sites, got %#v", issues)
	}
}

func TestLevel0AllowsThisInsideTraitMethods(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
trait HelperTrait {
    public function helper(): void
    {
        $this->local();
    }

    private function local(): void {}
}
`,
	})

	if hasIssueContaining(issues, level0SymbolsCode, "Undefined variable: $this") {
		t.Fatalf("trait methods should have object context, got %#v", issues)
	}
	if hasIssueContaining(issues, level0SymbolsCode, "Call to an undefined method HelperTrait::local") {
		t.Fatalf("trait methods should resolve other trait methods, got %#v", issues)
	}
}

func TestLevel0ResolvesNamespacedBuiltinClassFallback(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
namespace App;

function run(DateTimeImmutable $date): void
{
    $reflection = new ReflectionClass($date);
    DateTime::createFromFormat('Y-m-d', '2026-05-26');
}
`,
	})

	for _, unexpected := range []string{
		"Parameter $date references unknown class App\\DateTimeImmutable",
		"Instantiated class App\\ReflectionClass not found",
		"Call to an undefined static method App\\DateTime::createFromFormat",
	} {
		if hasIssueContaining(issues, level0SymbolsCode, unexpected) {
			t.Fatalf("expected builtin class fallback for %q, got %#v", unexpected, issues)
		}
	}
}

func TestLevel0ClassModelAndMethodChecks(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
final class Base {}
class Child extends Base {}
class UsesMissing implements MissingInterface {}
class Calls {
    public function run() {
        $this->missing();
    }
}
`,
	})

	if !hasIssueContaining(issues, level0ClassModelCode, "extends final class Base") {
		t.Fatalf("expected final class extension issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "implements unknown interface MissingInterface") {
		t.Fatalf("expected unknown interface issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Call to an undefined method Calls::missing") {
		t.Fatalf("expected undefined $this method issue, got %#v", issues)
	}
}

// TestLevel0ClassModelSpanIsHeaderOnly guards against regressing to
// underlining the entire class body: the diagnostic's span should end on
// the declaration line (after the extends clause), not on the closing '}'
// several lines later.
func TestLevel0ClassModelSpanIsHeaderOnly(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
final class Base {}
class Child extends Base
{
    public function run()
    {
        return 1;
    }
}
`,
	})

	var found *AnalysisIssue
	for i := range issues {
		if issues[i].Code == level0ClassModelCode && strings.Contains(issues[i].Message, "extends final class Base") {
			found = &issues[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected final class extension issue, got %#v", issues)
	}
	if found.Line != 3 {
		t.Fatalf("expected diagnostic to start on the declaration line 3, got line %d: %#v", found.Line, found)
	}
	if found.EndLine != 3 {
		t.Fatalf("expected diagnostic span to stay on the header line (3), got EndLine %d spanning into the body: %#v", found.EndLine, found)
	}
}

// TestLevel0InterfaceModelSpanIsHeaderOnly guards against underlining an
// entire interface body for a header-level diagnostic.
func TestLevel0InterfaceModelSpanIsHeaderOnly(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
interface Child extends Missing
{
    public function run(): void;

    public function stop(): void;
}
`,
	})

	var found *AnalysisIssue
	for i := range issues {
		if issues[i].Code == level0ClassModelCode && strings.Contains(issues[i].Message, "extends unknown interface Missing") {
			found = &issues[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected unknown interface extension issue, got %#v", issues)
	}
	if found.Line != 2 {
		t.Fatalf("expected diagnostic to start on the declaration line 2, got line %d: %#v", found.Line, found)
	}
	if found.EndLine != 2 {
		t.Fatalf("expected diagnostic span to stay on the header line (2), got EndLine %d spanning into the body: %#v", found.EndLine, found)
	}
}

// TestLevel0EnumModelSpanIsHeaderOnly guards against underlining an entire
// enum body for a header-level diagnostic.
func TestLevel0EnumModelSpanIsHeaderOnly(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
enum BadBacking: float
{
    case A;

    case B;
}
`,
	})

	var found *AnalysisIssue
	for i := range issues {
		if issues[i].Code == level0ClassModelCode && strings.Contains(issues[i].Message, "Backed enum BadBacking can have only") {
			found = &issues[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected backed enum type issue, got %#v", issues)
	}
	if found.Line != 2 {
		t.Fatalf("expected diagnostic to start on the declaration line 2, got line %d: %#v", found.Line, found)
	}
	if found.EndLine != 2 {
		t.Fatalf("expected diagnostic span to stay on the header line (2), got EndLine %d spanning into the body: %#v", found.EndLine, found)
	}
}

func TestLevel0ClassModelModifierLegality(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
final abstract class Impossible {}

class ConcreteWithAbstract {
    abstract public function missing();
}

abstract class BadAbstractMethods {
    abstract private function hidden();
    final abstract public function sealed();
}

class BadConstructor {
    public function __construct(): void {}
}

interface BadInterface {
    private function hidden();
}
`,
	})

	for _, expected := range []string{
		"Class Impossible cannot be both final and abstract",
		"Class ConcreteWithAbstract has abstract method missing() but is not abstract",
		"Abstract method BadAbstractMethods::hidden() cannot be private",
		"Abstract method BadAbstractMethods::sealed() cannot be final",
		"Constructor BadConstructor::__construct() cannot have a return type",
		"Interface method BadInterface::hidden() must be public",
	} {
		if !hasIssueContaining(issues, level0ClassModelCode, expected) {
			t.Fatalf("expected %q issue, got %#v", expected, issues)
		}
	}
}

func TestLevel0ConsistentConstructorTag(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
/**
 * @phpstan-consistent-constructor
 */
class Base {
    public function __construct(string $name) {}
}

class BadChild extends Base {
    protected function __construct(string $name, int $id) {}
}

/**
 * @phpstan-consistent-constructor
 */
class PrivateTagged {
    private function __construct() {}
}

/**
 * @phpstan-consistent-constructor
 */
final class FinalPrivateTagged {
    private function __construct() {}
}
`,
	})

	for _, expected := range []string{
		"Constructor BadChild::__construct() visibility must be at least as visible as Base::__construct()",
		"Method BadChild::__construct() requires more required parameters than the inherited method",
		"Class PrivateTagged has @phpstan-consistent-constructor but its constructor is private",
	} {
		if !hasIssueContaining(issues, level0ClassModelCode, expected) {
			t.Fatalf("expected %q issue, got %#v", expected, issues)
		}
	}
	if hasIssueContaining(issues, level0ClassModelCode, "FinalPrivateTagged has @phpstan-consistent-constructor") {
		t.Fatalf("final class with private consistent constructor should not be reported, got %#v", issues)
	}
}

func TestLevel0ClassModelRequiredMethodImplementations(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
interface RootContract {
    public function inheritedRequirement();
}

interface Contract extends RootContract {
    public function required();
    public function mustBePublic();
}

class MissingMethods implements Contract {
    public function inheritedRequirement() {}
}

class NonPublicImplementation implements Contract {
    public function inheritedRequirement() {}
    public function required() {}
    protected function mustBePublic() {}
}

abstract class AbstractBase {
    abstract public function fromParent();
}

class MissingParentMethod extends AbstractBase {}

class CompleteImplementation implements Contract {
    public function inheritedRequirement() {}
    public function required() {}
    public function mustBePublic() {}
}
`,
	})

	for _, expected := range []string{
		"Class MissingMethods must implement method required()",
		"Class MissingMethods must implement method mustBePublic()",
		"Method NonPublicImplementation::mustBePublic() implementing interface method must be public",
		"Class MissingParentMethod must implement method fromParent()",
	} {
		if !hasIssueContaining(issues, level0ClassModelCode, expected) {
			t.Fatalf("expected %q issue, got %#v", expected, issues)
		}
	}
	for _, unexpected := range []string{
		"Class CompleteImplementation must implement",
		"Class MissingMethods must implement method inheritedRequirement()",
	} {
		if hasIssueContaining(issues, level0ClassModelCode, unexpected) {
			t.Fatalf("unexpected %q issue, got %#v", unexpected, issues)
		}
	}
}

func TestLevel0ClassModelRequiredMethodSignatureCompatibility(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
interface SignatureContract {
    public function shape(string $name, int $count = 0): int;
}

class BadRequiredCount implements SignatureContract {
    public function shape(string $name, int $count, string $extra): int {}
}

class BadMaxCount implements SignatureContract {
    public function shape(string $name): int {}
}

class BadParamName implements SignatureContract {
    public function shape(string $label, int $count = 0): int {}
}

class BadReturn implements SignatureContract {
    public function shape(string $name, int $count = 0): string {}
}

class VariadicImplementation implements SignatureContract {
    public function shape(string $name, int $count = 0, ...$rest): int {}
}
`,
	})

	for _, expected := range []string{
		"Method BadRequiredCount::shape() requires more required parameters than the inherited method",
		"Method BadMaxCount::shape() accepts fewer parameters than the inherited method",
		"Parameter 1 of method BadParamName::shape() is named $label, expected $name",
		"Return type string of method BadReturn::shape() is not compatible with inherited return type int",
	} {
		if !hasIssueContaining(issues, level0ClassModelCode, expected) {
			t.Fatalf("expected %q issue, got %#v", expected, issues)
		}
	}
	if hasIssueContaining(issues, level0ClassModelCode, "VariadicImplementation::shape") {
		t.Fatalf("variadic compatible implementation should not be reported, got %#v", issues)
	}
}

func TestLevel0ClassModelIgnoresPHPDocOnlyAncestorReturnType(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
interface BootableServiceProviderInterface {
    /**
     * @return ServiceDefinition[]
     */
    public function boot(ContainerInterface $container);
}

class AccessControlModule implements BootableServiceProviderInterface {
    /**
     * @return ServiceDefinition[]|void
     */
    public function boot(ContainerInterface $container)
    {
    }
}
`,
	})

	if hasIssueContaining(issues, level0ClassModelCode, "AccessControlModule::boot") {
		t.Fatalf("PHPDoc-only ancestor return type must not be enforced as a native contract, got %#v", issues)
	}
}

func TestLevel0ClassModelReturnTypeCovariance(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
interface Entity {
    public function getId(): string|int|null;
}

class User implements Entity {
    public function getId(): ?string {
        return null;
    }
}

interface Handler {
    public function handle(): mixed;
}

class SpecificHandler implements Handler {
    public function handle(): string {
        return 'ok';
    }
}

interface ResponseHandler {
    public function response(): ?Response;
}

class Response {}
class ConcreteResponseHandler implements ResponseHandler {
    public function response(): Response {
        return new Response();
    }
}
`,
	})

	if hasIssueContaining(issues, level0ClassModelCode, "Return type ?string of method User::getId() is not compatible") {
		t.Fatalf("nullable subtype should satisfy union parent return, got %#v", issues)
	}
	if hasIssueContaining(issues, level0ClassModelCode, "Return type string of method SpecificHandler::handle() is not compatible") {
		t.Fatalf("specific return should satisfy mixed parent return, got %#v", issues)
	}
	if hasIssueContaining(issues, level0ClassModelCode, "Return type Response of method ConcreteResponseHandler::response() is not compatible") {
		t.Fatalf("non-null return should satisfy nullable parent return, got %#v", issues)
	}
}

func TestLevel0DuplicateDeclarationsAreReportedForOwningFile(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"a.php": `<?php
class Duplicate {}
`,
		"b.php": `<?php
class Duplicate {}
`,
		"c.php": `<?php
class Other {}
`,
	})

	if countIssueContaining(issues, level0ClassModelCode, "Duplicate declaration of class Duplicate") != 1 {
		t.Fatalf("expected one duplicate declaration issue, got %#v", issues)
	}
	for _, issue := range issues {
		if strings.Contains(issue.Message, "Duplicate declaration of class Duplicate") && issue.Filename == "c.php" {
			t.Fatalf("duplicate issue reported for unrelated file: %#v", issues)
		}
	}
}

func TestClassModelAncestorChecksUseResolverWithoutProjectMaps(t *testing.T) {
	const filename = "model.php"
	nodes := parsePHPForProjectIndex(t, `<?php
class BaseModel {
    final public const KIND = "base";
    final public function locked(): void {}
}

class ChildModel extends BaseModel {
    public const KIND = "child";
    public function locked(): void {}
}
`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	ctx := snapshot.NewAnalysisContext()
	issues := (&Level0Rule{}).checkClassModel(filename, nodes, ctx, CollectFileTypeContext(nodes))
	if !hasIssueContaining(issues, level0ClassModelCode, "Cannot override final method BaseModel::locked") {
		t.Fatalf("expected resolver-backed final method diagnostic, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "Cannot override final constant BaseModel::KIND") {
		t.Fatalf("expected resolver-backed final constant diagnostic, got %#v", issues)
	}
}

func TestLevel0TypeUseCatchAndAttributeReferences(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
use Missing\Thing;
use function missing_fn;
use const MISSING_CONST;

#[MissingAttr]
function demo(MissingParam $value): MissingReturn {}

class Box {
    private MissingProperty $item;
}

try {
} catch (MissingException $e) {
}
`,
	})

	for _, expected := range []string{
		"Used function missing_fn not found",
		"Used constant MISSING_CONST not found",
		"Attribute class MissingAttr not found",
		"Parameter $value references unknown class MissingParam",
		"Return type references unknown class MissingReturn",
		"Property $item references unknown class MissingProperty",
		"Caught class MissingException not found",
	} {
		if !hasIssueContaining(issues, level0SymbolsCode, expected) {
			t.Fatalf("expected %q issue, got %#v", expected, issues)
		}
	}
}

func TestLevel0PropertyChecks(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Props {
    public int $known;
    public int $instance;
    public static int $staticKnown;

    public function run() {
        $this->missing;
        self::$missingStatic;
        self::$instance;
    }
}
`,
	})

	if !hasIssueContaining(issues, level0SymbolsCode, "Access to an undefined property Props::$missing") {
		t.Fatalf("expected undefined instance property issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Access to undefined static property Props::$missingStatic") {
		t.Fatalf("expected undefined static property issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Static access to instance property Props::$instance") {
		t.Fatalf("expected static access to instance property issue, got %#v", issues)
	}
}

func TestLevel1IssetAndEmptyAllowUndefinedVariables(t *testing.T) {
	issues := runLevel1OnFiles(t, map[string]string{
		"test.php": `<?php
isset($missing);
empty($alsoMissing);
echo $reported;
`,
	})

	if hasIssueContaining(issues, level1VariablesCode, "Variable $missing might not be defined.") ||
		hasIssueContaining(issues, level1VariablesCode, "Variable $alsoMissing might not be defined.") {
		t.Fatalf("isset/empty variables should not be reported, got %#v", issues)
	}
	if !hasIssueContaining(issues, level1VariablesCode, "Variable $reported might not be defined.") {
		t.Fatalf("expected normal undefined variable issue, got %#v", issues)
	}
}

func TestLevel0ReflectionGuardsSuppressTypeAndConstantReferences(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
use Missing\GuardedClass;
use function missing_guarded_fn;
use const Missing\GUARDED_CONST;

if (interface_exists(Missing\GuardedClass::class)) {}
if (trait_exists('Missing\\GuardedTrait')) {}
if (function_exists('missing_guarded_fn')) {}
if (defined('Missing\\GUARDED_CONST')) {}
echo Missing\STILL_MISSING_CONST;
`,
	})

	for _, unexpected := range []string{
		"Used class Missing\\GuardedClass not found",
		"Used function missing_guarded_fn not found",
		"Used constant Missing\\GUARDED_CONST not found",
	} {
		if hasIssueContaining(issues, level0SymbolsCode, unexpected) {
			t.Fatalf("reflection guard should suppress %q, got %#v", unexpected, issues)
		}
	}
}

func TestLevel0DefinedGuardSuppressesClassConstantAccess(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
if (defined('Missing\\Guarded::VALUE')) {
    echo Missing\Guarded::VALUE;
}
echo Missing\StillMissing::VALUE;
`,
	})

	if hasIssueContaining(issues, level0SymbolsCode, "Access to undefined constant Missing\\Guarded::VALUE") {
		t.Fatalf("defined() guard should suppress guarded constant access, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Missing\\StillMissing") {
		t.Fatalf("expected unguarded class constant issue, got %#v", issues)
	}
}

func TestLevel0ExcludesLevel3ThrowTypeChecks(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
throw new Exception();
throw new DateTime();
throw new MissingThrowable();
`,
	})

	if hasIssueContaining(issues, level3ThrowTypeCode, "to throw") {
		t.Fatalf("level zero should exclude level-three throw diagnostics, got %#v", issues)
	}
}

func TestLevel0ReadonlyClassInheritance(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"a.php": `<?php
class MutableParent {}
readonly class ReadonlyParent {}
`,
		"b.php": `<?php
readonly class ReadonlyChild extends MutableParent {}
class MutableChild extends ReadonlyParent {}
`,
	})

	if !hasIssueContaining(issues, level0ClassModelCode, "Readonly class ReadonlyChild cannot extend non-readonly class MutableParent") {
		t.Fatalf("expected readonly extends mutable issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "Non-readonly class MutableChild cannot extend readonly class ReadonlyParent") {
		t.Fatalf("expected mutable extends readonly issue, got %#v", issues)
	}
}

func TestLevel0ThisInStaticMethod(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Demo {
    public static function run() {
        $this->work();
        $this->value;
    }
    public function work() {}
    public int $value;
}
`,
	})

	for _, expected := range []string{
		"Using $this inside static method Demo::run()",
	} {
		if countIssueContaining(issues, level0SymbolsCode, expected) < 2 {
			t.Fatalf("expected at least two %q issues, got %#v", expected, issues)
		}
	}
}

func TestLevel0ConstructorVisibilityAndArgumentCount(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class PrivateCtor {
    private function __construct() {}
}
class ProtectedCtor {
    protected function __construct() {}
}
class Child extends ProtectedCtor {
    public static function create() {
        new Child();
    }
}
new PrivateCtor();
new ProtectedCtor();
new Child();
new NoCtor(1);
class NoCtor {}
`,
	})

	for _, expected := range []string{
		"Cannot instantiate class PrivateCtor via private constructor",
		"Cannot instantiate class ProtectedCtor via protected constructor",
		"Cannot instantiate class Child via protected constructor",
		"Class NoCtor constructor invoked with 1",
	} {
		if !hasIssueContaining(issues, level0InvocationCode, expected) {
			t.Fatalf("expected %q issue, got %#v", expected, issues)
		}
	}
	if countIssueContaining(issues, level0InvocationCode, "Cannot instantiate class Child via protected constructor") != 1 {
		t.Fatalf("global and static-context Child instantiation should report once, got %#v", issues)
	}
}

func TestLevel0InstanceCallToStaticMethod(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Demo {
    public static function run() {}
}
(new Demo())->run();
`,
	})

	if hasIssueContaining(issues, level0InvocationCode, "Call to static method Demo::run() on instance") {
		t.Fatalf("PHPStan does not report instance syntax for a static method, got %#v", issues)
	}
}

func TestLevel0ProtectedMethodCallableFromSubclass(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Base {
    protected function work() {}
}
class Child extends Base {
    public function run() {
        $this->work();
    }
}
`,
	})

	if hasIssueContaining(issues, level0InvocationCode, "Call to protected method") {
		t.Fatalf("protected method should be callable from subclass, got %#v", issues)
	}
}

func TestLevel0ReadonlyClassPropertiesAreImplicitlyReadonly(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
readonly class ValueObject {
    public int $id;
}
readonly class ReadonlyParent {
    public readonly int $inherited;
}
readonly class Child extends ReadonlyParent {
    public int $inherited;
}
`,
	})

	if hasIssueContaining(issues, level0ClassModelCode, "cannot have non-readonly property") {
		t.Fatalf("readonly class properties are implicitly readonly, got %#v", issues)
	}
	if hasIssueContaining(issues, level0ClassModelCode, "overriding readonly property must be readonly") {
		t.Fatalf("readonly child class properties are implicitly readonly, got %#v", issues)
	}
}

func TestLevel0EnumSanity(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
enum UnitWithValue {
    case A = 1;
}

enum BadBacking: float {
    case A;
}

enum MissingBackedValue: string {
    case A;
}

enum BadUnitValue {
    case A = 'x';
}

enum BadCaseType: string {
    case A = 1;
}

enum DuplicateValues: int {
    case A = 1;
    case B = 1;
}

enum BadMethods: string {
    case A = 'a';
    public function __construct() {}
    public function __destruct() {}
    public function __sleep() {}
    public function cases() {}
    public static function from(string $value): self {}
}

enum BadSerializable implements \Serializable {
    case A;
}

enum ValidBacked: string {
    case A = 'a';
    public function label(): string { return $this->name; }
}
`,
	})

	for _, expected := range []string{
		"Enum UnitWithValue is not backed, but case A has value 1",
		"Backed enum BadBacking can have only \"int\" or \"string\" type",
		"Enum case MissingBackedValue::A does not have a value but the enum is backed with the \"string\" type",
		"Enum BadUnitValue is not backed, but case A has value \"x\"",
		"Enum case BadCaseType::A value 1 does not match the \"string\" type",
		"Enum DuplicateValues has duplicate value 1 for cases A, B",
		"Enum BadMethods contains constructor",
		"Enum BadMethods contains destructor",
		"Enum BadMethods contains magic method __sleep()",
		"Enum BadMethods cannot redeclare native method cases()",
		"Enum BadMethods cannot redeclare native method from()",
		"Enum BadSerializable cannot implement Serializable",
	} {
		if !hasIssueContaining(issues, level0ClassModelCode, expected) {
			t.Fatalf("expected %q issue, got %#v", expected, issues)
		}
	}
	for _, unexpected := range []string{
		"Enum ValidBacked",
	} {
		if hasIssueContaining(issues, level0ClassModelCode, unexpected) {
			t.Fatalf("unexpected %q issue, got %#v", unexpected, issues)
		}
	}
}

func TestLevel0FinalMethodOverride(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Base {
    final public function sealed() {}
}
class Child extends Base {
    public function sealed() {}
}
`,
	})

	if !hasIssueContaining(issues, level0ClassModelCode, "Cannot override final method Base::sealed()") {
		t.Fatalf("expected final method override issue, got %#v", issues)
	}
}

func TestLevel0PrivateMethodNotCallableFromSubclass(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Base {
    private function hidden() {}
}
class Child extends Base {
    public function run() {
        $this->hidden();
    }
}
`,
	})

	if !hasIssueContaining(issues, level0InvocationCode, "Call to private method Base::hidden()") {
		t.Fatalf("expected private method visibility issue, got %#v", issues)
	}
}

func TestLevel0ProtectedMethodOnKnownReceiver(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Base {
    protected function work() {}
}
(new Base())->work();
`,
	})

	if hasIssueContaining(issues, level2MethodVisibilityCode, "Call to protected method Base::work()") {
		t.Fatalf("level zero should exclude level-two protected visibility checks, got %#v", issues)
	}
}

func TestLevel0ReflectionGuardsSuppressUnknownSymbols(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
if (class_exists(MissingGuarded::class)) {
    new MissingGuarded();
}
if (function_exists('guarded_function')) {
    guarded_function();
}
class GuardedMethods {
    public function run() {
        if (method_exists($this, 'guardedMethod')) {
            $this->guardedMethod();
        }
    }
}
new StillMissing();
unguarded_function();
`,
	})

	for _, unexpected := range []string{
		"MissingGuarded not found",
		"Function guarded_function not found",
		"Call to an undefined method GuardedMethods::guardedMethod",
	} {
		if hasIssueContaining(issues, level0SymbolsCode, unexpected) {
			t.Fatalf("reflection guard should suppress %q, got %#v", unexpected, issues)
		}
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Instantiated class StillMissing not found") {
		t.Fatalf("expected unguarded class issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Function unguarded_function not found") {
		t.Fatalf("expected unguarded function issue, got %#v", issues)
	}
}

func TestLevel0ClassConstantVisibility(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Base {
    private const SECRET = 1;
    protected const TOKEN = 2;
    public const NAME = 'base';
}
class Child extends Base {
    public function ok() {
        return self::TOKEN;
    }
}
echo Base::SECRET;
echo Base::TOKEN;
echo Child::NAME;
`,
	})

	if !hasIssueContaining(issues, level0SymbolsCode, "Access to private constant Base::SECRET") {
		t.Fatalf("expected private constant issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0SymbolsCode, "Access to protected constant Base::TOKEN") {
		t.Fatalf("expected protected constant issue, got %#v", issues)
	}
	if hasIssueContaining(issues, level0SymbolsCode, "Access to undefined constant Child::NAME") {
		t.Fatalf("inherited public constant should resolve, got %#v", issues)
	}
}

func TestSymbolAndInvocationChecksUseResolverWithoutProjectIndex(t *testing.T) {
	const filename = "test.php"
	nodes := parsePHPForProjectIndex(t, `<?php
trait Helpers {
    private function privateHelper(): void {}
}
class Base {
    private const SECRET = 1;
    protected const TOKEN = 2;
    protected function __construct() {}
    private function hidden(): void {}
    protected function work(): void {}
}
class Child extends Base {
    use Helpers;
    public function run(): void {
        new Child();
        $this->work();
        self::TOKEN;
        Base::hidden();
        Base::SECRET;
    }
}
new Child();
`)
	snapshot, err := NewSemanticSnapshot(map[string][]ast.Node{filename: nodes}, nil)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	ctx := snapshot.NewAnalysisContext()
	fileCtx := analysisFileTypeContext(ctx, nodes)
	issues := (&Level0Rule{}).checkSymbolsAndCalls(filename, nodes, ctx, fileCtx)

	for _, expected := range []struct {
		code    string
		message string
	}{
		{level0InvocationCode, "Cannot instantiate class Child via protected constructor"},
		{level0InvocationCode, "Call to private method Base::hidden()"},
		{level0SymbolsCode, "Access to private constant Base::SECRET"},
	} {
		if !hasIssueContaining(issues, expected.code, expected.message) {
			t.Fatalf("expected resolver-only issue %q, got %#v", expected.message, issues)
		}
	}
	for _, unexpected := range []struct {
		code    string
		message string
	}{
		{level0InvocationCode, "Call to protected method Base::work()"},
		{level0SymbolsCode, "Access to protected constant Base::TOKEN"},
	} {
		if hasIssueContaining(issues, unexpected.code, unexpected.message) {
			t.Fatalf("unexpected resolver-only issue %q, got %#v", unexpected.message, issues)
		}
	}

	child := nodes[2].(*ast.ClassNode)
	var traitVisibilityIssues []AnalysisIssue
	checkMethodVisibility(filename, child.GetPos(), ResolvedMethod{Name: "privateHelper", DeclaringClass: "Helpers", Visibility: "private"}, "Child", child, fileCtx, ctx.Resolver, false, &traitVisibilityIssues)
	if len(traitVisibilityIssues) != 0 {
		t.Fatalf("trait visibility should resolve without project maps, got %#v", traitVisibilityIssues)
	}
}

func TestLevel0InterfaceConstantsMustBePublic(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
interface BadConstants {
    private const SECRET = 1;
    protected const TOKEN = 2;
    public const NAME = 'ok';
}
`,
	})

	if !hasIssueContaining(issues, level0ClassModelCode, "Interface constant BadConstants::SECRET must be public") {
		t.Fatalf("expected private interface constant issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "Interface constant BadConstants::TOKEN must be public") {
		t.Fatalf("expected protected interface constant issue, got %#v", issues)
	}
	if hasIssueContaining(issues, level0ClassModelCode, "BadConstants::NAME must be public") {
		t.Fatalf("public interface constant should not be reported, got %#v", issues)
	}
}

func TestLevel0FinalClassConstantLegality(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
class Base {
    final public const LOCKED = 1;
    private final const SECRET = 2;
}
class Child extends Base {
    public const LOCKED = 3;
}
`,
	})

	if !hasIssueContaining(issues, level0ClassModelCode, "Cannot override final constant Base::LOCKED") {
		t.Fatalf("expected final constant override issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "Private constant Base::SECRET cannot be final") {
		t.Fatalf("expected private final constant issue, got %#v", issues)
	}
}

func TestLevel1CompactReportsUndefinedVariables(t *testing.T) {
	issues := runLevel1OnFiles(t, map[string]string{
		"test.php": `<?php
$defined = 1;
compact('defined', 'missing');
`,
	})

	if hasIssueContaining(issues, level1VariablesCode, "Variable $defined might not be defined.") {
		t.Fatalf("defined compact variable should not be reported, got %#v", issues)
	}
	if !hasIssueContaining(issues, level1VariablesCode, "Variable $missing might not be defined.") {
		t.Fatalf("expected compact undefined variable issue, got %#v", issues)
	}
}

func TestLevel0CrossFileIndexResolvesKnownSymbols(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"a.php": `<?php
namespace App;
class Service {
    public function work($a) {}
}
function helper($a) {}
`,
		"b.php": `<?php
namespace App;
$s = new Service();
$s->work(1);
helper(1);
`,
	})

	for _, issue := range issues {
		if strings.Contains(issue.Message, "Service not found") || strings.Contains(issue.Message, "helper not found") {
			t.Fatalf("expected cross-file symbols to resolve, got %#v", issues)
		}
	}
}

func TestLevel0FullyQualifiedStaticCallIsNotRenamespaced(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"global.php": `<?php
class RG_DB_DBFieldType {
    const _STR = 1;
    public static function MysqlEscape($type, $value) {}
}
`,
		"namespaced.php": `<?php
namespace RG\Utility\GeoHash;

function run($value) {
    \RG_DB_DBFieldType::MysqlEscape(\RG_DB_DBFieldType::_STR, $value);
}
`,
	})

	for _, issue := range issues {
		if strings.Contains(issue.Message, "RG_DB_DBFieldType") {
			t.Fatalf("expected fully-qualified static call to resolve to global class, got %#v", issues)
		}
	}
}

func TestLevel0LanguageChecksExcludeLevel1Variables(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
echo $missing;
goto nowhere;
$a = ['x' => 1, 'x' => 2];
`,
	})

	if hasIssueContaining(issues, level1VariablesCode, "Variable $missing might not be defined.") {
		t.Fatalf("level zero should exclude undefined-variable diagnostics, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0LanguageCode, "Goto to undefined label nowhere") {
		t.Fatalf("expected undefined label issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0LanguageCode, "duplicate key") {
		t.Fatalf("expected duplicate array key issue, got %#v", issues)
	}
}

func TestLevel0InvalidVoidAndUnsetCasts(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
$x = (void) 1;
$y = (unset) 1;
`,
	})
	if !hasIssueContaining(issues, level0LanguageCode, "Cannot cast to void.") {
		t.Fatalf("expected void cast issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0LanguageCode, "Cannot cast to unset.") {
		t.Fatalf("expected unset cast issue, got %#v", issues)
	}
}

func TestAnalysisLevel0DoesNotRunHigherLevelRules(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
function bad(): int {
    return "x";
}
`,
	})

	for _, issue := range issues {
		if issue.Code == "A.RETURN.TYPE" || issue.Code == "A.PROP.TYPE" || issue.Code == "A.ARG.TYPE" || issue.Code == "Generic.CodeAnalysis.UnreachableCode" {
			t.Fatalf("higher-level issue emitted at level 0: %#v", issues)
		}
	}
}

// TestLevel0CyclicClassHierarchyDoesNotHang guards against unbounded recursion
// in ancestor-walking helpers (finalMethodInAncestors, finalConstantInAncestors,
// consistentConstructorInAncestors, collectAbstractMethods,
// collectUnimplementedParentAbstractMethods, isSubclassOf). Indexed PHP source
// can legally parse into a self-referential or mutually cyclic "extends" chain
// even though PHP itself would reject it at runtime; without cycle detection
// these helpers previously recursed forever and crashed the analyser with a
// stack overflow on large, real-world corpora.
func TestLevel0CyclicClassHierarchyDoesNotHang(t *testing.T) {
	done := make(chan []AnalysisIssue, 1)
	go func() {
		done <- runLevel0OnFiles(t, map[string]string{
			"self.php": `<?php
class SelfLoop extends SelfLoop {
    final public function foo(): void {}
    final public const BAR = 1;
}
`,
			"mutual_a.php": `<?php
class MutualA extends MutualB {
    public function useOther(): void {}
}
`,
			"mutual_b.php": `<?php
class MutualB extends MutualA implements MutualIface {
    abstract public function foo(): void;
}
interface MutualIface extends MutualIface {
}
`,
		})
	}()

	select {
	case <-done:
		// Completed without hanging or stack-overflowing.
	case <-time.After(5 * time.Second):
		t.Fatal("analysis did not complete: likely unbounded recursion on cyclic class hierarchy")
	}
}
