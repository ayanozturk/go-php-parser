package analyse

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/ast"
)

func hasPropertyTypeIssue(issues []AnalysisIssue) bool {
	for _, iss := range issues {
		if iss.Code == "A.PROP.TYPE" {
			return true
		}
	}
	return false
}

func hasAssignOpInvalidIssue(issues []AnalysisIssue) bool {
	for _, issue := range issues {
		if issue.Code == assignOpInvalidCode {
			return true
		}
	}
	return false
}

func TestPropertyAssignmentTypeMismatch(t *testing.T) {
	php := `<?php
    class Example {
        private int $count;

        public function run(): void {
            $this->count = "bad";
        }
    }`
	issues := analysePHP(t, php)
	if !hasPropertyTypeIssue(issues) {
		t.Fatalf("expected A.PROP.TYPE issue for string assigned to int property, got: %#v", issues)
	}
}

func TestPropertyAssignmentTypeCompatible(t *testing.T) {
	php := `<?php
    class Example {
        private float $count;

        public function run(): void {
            $this->count = 1;
        }
    }`
	issues := analysePHP(t, php)
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("expected no A.PROP.TYPE issue for int assigned to float property, got: %#v", issues)
	}
}

func TestStaticPropertyAssignmentTypeMismatch(t *testing.T) {
	php := `<?php
    class Example {
        private static int $count;

        public static function run(): void {
            self::$count = "bad";
        }
    }`
	issues := analysePHP(t, php)
	if !hasPropertyTypeIssue(issues) {
		t.Fatalf("expected A.PROP.TYPE issue for string assigned to static int property, got: %#v", issues)
	}
}

func TestInvalidCompoundPropertyAssignmentUsesOperationDiagnostic(t *testing.T) {
	php := `<?php
    class Example {
        private int $count;

        public function run(): void {
            $this->count += "bad";
        }
    }`
	issues := analysePHP(t, php)
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("compound assignment requires operation-result analysis, got misleading property issue: %#v", issues)
	}
	if !hasAssignOpInvalidIssue(issues) {
		t.Fatalf("expected %s for invalid compound assignment, got: %#v", assignOpInvalidCode, issues)
	}
}

func TestCompoundAssignmentResultMatrix(t *testing.T) {
	tests := []struct {
		name     string
		operator string
		left     string
		right    string
		result   string
		valid    bool
		known    bool
	}{
		{name: "integer addition", operator: "+=", left: "int", right: "int", result: "int", valid: true, known: true},
		{name: "float promotion", operator: "+=", left: "int", right: "float", result: "float", valid: true, known: true},
		{name: "array union", operator: "+=", left: "array", right: "array", result: "array", valid: true, known: true},
		{name: "invalid numeric string", operator: "+=", left: "int", right: "string", valid: false, known: true},
		{name: "concatenation", operator: ".=", left: "int", right: "string", result: "string", valid: true, known: true},
		{name: "unsupported boolean arithmetic", operator: "+=", left: "bool", right: "int", valid: false, known: false},
		{name: "coalesce assignment handled elsewhere", operator: "??=", left: "int", right: "int", valid: false, known: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, valid, known := compoundAssignmentResult(test.operator, ParseType(test.left), ParseType(test.right))
			if valid != test.valid || known != test.known || result.String() != test.result {
				t.Fatalf("result = (%s, %t, %t), want (%s, %t, %t)", result.String(), valid, known, test.result, test.valid, test.known)
			}
		})
	}
}

func TestPropertyAssignmentAcceptsImplementedInterface(t *testing.T) {
	php := `<?php
	namespace Doctrine\Common\Collections;

	interface Collection {}

	class ArrayCollection implements Collection {}

	class Example {
		private Collection $users;

		public function __construct()
		{
			$this->users = new ArrayCollection();
		}
	}`
	issues := analysePHP(t, php)
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("expected no A.PROP.TYPE issue for class implementing interface assignment, got: %#v", issues)
	}
}

func TestPropertyTypeRuleUsesSemanticInferredTypeFact(t *testing.T) {
	const filename = "src/Example.php"
	nodes, rhs := parsePropertyAssignmentFactFixture(t, `<?php
class Example {
    private int $count;

    public function run(): void {
        $this->count = "bad";
    }
}
`)
	key := inferredTypeFactKey(filename, rhs)
	reader := &countingFactReader{facts: map[SemanticFactKey]SemanticFact{key: {Key: key, Type: "int"}}}
	project := BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	ctx := &AnalysisContext{Resolver: project, Facts: reader}

	issues := (&PropertyTypeRule{}).CheckIssues(nodes, filename, ctx)
	if hasPropertyTypeIssue(issues) {
		t.Fatalf("expected inferred-type fact to satisfy int property, got: %#v", issues)
	}
}

func TestPropertyTypeRuleFallsBackForNonmatchingSemanticFact(t *testing.T) {
	const filename = "src/Example.php"
	nodes, rhs := parsePropertyAssignmentFactFixture(t, `<?php
class Example {
    private int $count;

    public function run(): void {
        $this->count = "bad";
    }
}
`)
	key := inferredTypeFactKey(filename, rhs)
	key.EndOffset++
	reader := &countingFactReader{facts: map[SemanticFactKey]SemanticFact{key: {Key: key, Type: "int"}}}
	project := BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	ctx := &AnalysisContext{Resolver: project, Facts: reader}

	issues := (&PropertyTypeRule{}).CheckIssues(nodes, filename, ctx)
	if !hasPropertyTypeIssue(issues) {
		t.Fatalf("expected fallback inference to report string assigned to int, got: %#v", issues)
	}
}

func TestPropertyTypeRuleUsesSemanticReceiverTypeFact(t *testing.T) {
	const filename = "src/Example.php"
	nodes := parsePHPForProjectIndex(t, `<?php
class Holder {
    public int $count;
}

class Example {
    public function run(): void {
        $holder->count = "bad";
    }
}
`)
	class := nodes[1].(*ast.ClassNode)
	method := class.Methods[0].(*ast.FunctionNode)
	assignment := method.Body[0].(*ast.ExpressionStmt).Expr.(*ast.AssignmentNode)
	receiver := assignment.Left.(*ast.PropertyFetchNode).Object
	key := inferredTypeFactKey(filename, receiver)
	reader := &countingFactReader{facts: map[SemanticFactKey]SemanticFact{key: {Key: key, Type: "Holder"}}}
	project := BuildProjectIndex(map[string][]ast.Node{filename: nodes})
	ctx := &AnalysisContext{Resolver: project, Facts: reader}

	issues := (&PropertyTypeRule{}).CheckIssues(nodes, filename, ctx)
	if !hasPropertyTypeIssue(issues) {
		t.Fatalf("expected receiver fact to resolve Holder::$count mismatch, got: %#v", issues)
	}
}

func parsePropertyAssignmentFactFixture(t *testing.T, source string) ([]ast.Node, ast.Node) {
	t.Helper()
	nodes := parsePHPForProjectIndex(t, source)
	class, ok := nodes[0].(*ast.ClassNode)
	if !ok || len(class.Methods) != 1 {
		t.Fatalf("expected class with one method, got %#v", nodes)
	}
	method, ok := class.Methods[0].(*ast.FunctionNode)
	if !ok || len(method.Body) == 0 {
		t.Fatalf("expected method with statements, got %#v", class.Methods[0])
	}
	statement, ok := method.Body[len(method.Body)-1].(*ast.ExpressionStmt)
	if !ok {
		t.Fatalf("expected final expression statement, got %#v", method.Body[len(method.Body)-1])
	}
	assignment, ok := statement.Expr.(*ast.AssignmentNode)
	if !ok || assignment.Right == nil {
		t.Fatalf("expected property assignment, got %#v", statement.Expr)
	}
	return nodes, assignment.Right
}
