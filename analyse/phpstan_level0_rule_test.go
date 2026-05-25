package analyse

import (
	"go-phpcs/ast"
	"go-phpcs/lexer"
	"go-phpcs/parser"
	"strings"
	"testing"
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
	t.Helper()
	parsed := make(map[string][]ast.Node, len(files))
	for filename, php := range files {
		parsed[filename] = parsePHPForLevel0(t, php)
	}
	project := BuildProjectIndex(parsed)
	level := 0
	var issues []AnalysisIssue
	for filename, nodes := range parsed {
		ctx := &AnalysisContext{Resolver: project, Project: project, AnalysisLevel: &level}
		issues = append(issues, RunAnalysisRulesWithContext(filename, nodes, ctx)...)
	}
	return issues
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
		"Used class Missing\\Thing not found",
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

func TestLevel0IssetAndEmptyAllowUndefinedVariables(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
isset($missing);
empty($alsoMissing);
echo $reported;
`,
	})

	if hasIssueContaining(issues, level0VariablesCode, "Undefined variable: $missing") ||
		hasIssueContaining(issues, level0VariablesCode, "Undefined variable: $alsoMissing") {
		t.Fatalf("isset/empty variables should not be reported, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0VariablesCode, "Undefined variable: $reported") {
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

func TestLevel0ThrowExpressionValidity(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
throw new Exception();
throw new DateTime();
throw new MissingThrowable();
`,
	})

	if hasIssueContaining(issues, level0ClassModelCode, "Invalid type Exception to throw") {
		t.Fatalf("Exception should be throwable, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "Invalid type DateTime to throw") {
		t.Fatalf("expected non-throwable throw issue, got %#v", issues)
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

	if !hasIssueContaining(issues, level0InvocationCode, "Call to static method Demo::run() on instance") {
		t.Fatalf("expected instance static method call issue, got %#v", issues)
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

func TestLevel0ReadonlyClassProperties(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
readonly class BadProperty {
    public int $mutable;
}
readonly class ReadonlyParent {
    public readonly int $inherited;
}
readonly class BadOverride extends ReadonlyParent {
    public int $inherited;
}
`,
	})

	if !hasIssueContaining(issues, level0ClassModelCode, "Readonly class BadProperty cannot have non-readonly property $mutable") {
		t.Fatalf("expected readonly class property issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0ClassModelCode, "overriding readonly property must be readonly") {
		t.Fatalf("expected readonly override issue, got %#v", issues)
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

	if !hasIssueContaining(issues, level0InvocationCode, "Call to protected method Base::work()") {
		t.Fatalf("expected protected method call on known receiver, got %#v", issues)
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

func TestLevel0CompactReportsUndefinedVariables(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
$defined = 1;
compact('defined', 'missing');
`,
	})

	if hasIssueContaining(issues, level0VariablesCode, "Undefined variable: $defined") {
		t.Fatalf("defined compact variable should not be reported, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0VariablesCode, "Undefined variable: $missing") {
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

func TestLevel0UndefinedVariablesAndLanguageChecks(t *testing.T) {
	issues := runLevel0OnFiles(t, map[string]string{
		"test.php": `<?php
echo $missing;
goto nowhere;
$a = ['x' => 1, 'x' => 2];
`,
	})

	if !hasIssueContaining(issues, level0VariablesCode, "Undefined variable: $missing") {
		t.Fatalf("expected undefined variable issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0LanguageCode, "Goto to undefined label nowhere") {
		t.Fatalf("expected undefined label issue, got %#v", issues)
	}
	if !hasIssueContaining(issues, level0LanguageCode, "duplicate key") {
		t.Fatalf("expected duplicate array key issue, got %#v", issues)
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
