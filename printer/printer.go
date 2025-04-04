package printer

import (
	"fmt"
	"go-phpcs/ast"
	"io"
	"os"
	"strings"
)

// Printer holds the state for AST printing
type Printer struct {
	w      io.Writer
	indent int
}

// New creates a new Printer that writes to the given writer
func New(w io.Writer) *Printer {
	if w == nil {
		w = os.Stdout
	}
	return &Printer{w: w}
}

// PrintAST prints the AST nodes with proper indentation
func PrintAST(nodes []ast.Node, indent int) {
	p := New(os.Stdout)
	p.indent = indent
	p.printNodes(nodes)
}

func (p *Printer) printf(format string, args ...interface{}) {
	fmt.Fprintf(p.w, format, args...)
}

func (p *Printer) printIndent() {
	for i := 0; i < p.indent; i++ {
		p.printf("  ")
	}
}

func (p *Printer) printNodes(nodes []ast.Node) {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		p.printf(node.String() + "\n")
		// p.printNode(node)
	}
}

func (p *Printer) printNode(node ast.Node) {
	p.printIndent()
	p.printNodeType(node)
	p.printf(" @ %d:%d\n", node.GetPos().Line, node.GetPos().Column)

	p.indent++
	defer func() { p.indent-- }()

	switch n := node.(type) {
	case *ast.ArrayNode:
		p.printArray(n)
	case *ast.CommentNode:
		p.printIndent()
		p.printf("Value: %s\n", n.Value)
	case *ast.ArrayItemNode:
		p.printArrayItem(n)
	case *ast.FunctionNode:
		p.printFunction(n)
	case *ast.AssignmentNode:
		p.printAssignment(n)
	case *ast.BinaryExpr:
		p.printBinaryExpr(n)
	case *ast.ReturnNode:
		p.printReturn(n)
	case *ast.ExpressionStmt:
		p.printExpressionStmt(n)
	case *ast.IfNode:
		p.printIf(n)
	case *ast.WhileNode:
		p.printWhile(n)
	case *ast.InterpolatedStringLiteral:
		p.printInterpolatedString(n)
	case *ast.ClassNode:
		p.printClass(n)
	case *ast.EnumNode:
		p.printEnum(n)
	case *ast.VariableNode:
		p.printVariable(n)
	case *ast.StringLiteral:
		p.printStringLiteral(n)
	case *ast.IntegerLiteral:
		p.printIntegerLiteral(n)
	case *ast.FloatLiteral:
		p.printFloatLiteral(n)
	case *ast.BooleanLiteral:
		p.printBooleanLiteral(n)
	case *ast.NullLiteral:
		p.printNullLiteral(n)
	}
}

func (p *Printer) printNodeType(node ast.Node) {
	p.printf("%s", node.NodeType())
}

func (p *Printer) printArray(n *ast.ArrayNode) {
	if len(n.Elements) > 0 {
		p.printIndent()
		elements := make([]string, len(n.Elements))
		for i, elem := range n.Elements {
			if item, ok := elem.(*ast.ArrayItemNode); ok {
				elements[i] = p.arrayItemToString(item)
			}
		}
		p.printf("[ %s ]\n", strings.Join(elements, ", "))
	} else {
		p.printIndent()
		p.printf("[]\n")
	}
}

func (p *Printer) arrayItemToString(item *ast.ArrayItemNode) string {
	var result string
	if item.ByRef {
		result += "&"
	}
	if item.Unpack {
		result += "..."
	}
	if item.Key != nil {
		result += fmt.Sprintf("%s => ", item.Key.TokenLiteral())
	}
	result += item.Value.TokenLiteral()
	return result
}

func (p *Printer) printArrayItem(n *ast.ArrayItemNode) {
	if n.Key != nil {
		p.printIndent()
		p.printf("Key:\n")
		p.printNode(n.Key)
	}
	p.printIndent()
	p.printf("Value:\n")
	p.printNode(n.Value)
	if n.ByRef {
		p.printIndent()
		p.printf("ByRef: true\n")
	}
	if n.Unpack {
		p.printIndent()
		p.printf("Unpack: true\n")
	}
}

func (p *Printer) printFunction(n *ast.FunctionNode) {
	if n.Name != "" {
		p.printIndent()
		p.printf("Name: %s\n", n.Name)
	}
	if n.Visibility != "" {
		p.printIndent()
		p.printf("Visibility: %s\n", n.Visibility)
	}
	if n.ReturnType != "" {
		p.printIndent()
		p.printf("ReturnType: %s\n", n.ReturnType)
	}
	if len(n.Params) > 0 {
		p.printIndent()
		p.printf("Parameters:\n")
		p.printNodes(n.Params)
	}
	if len(n.Body) > 0 {
		p.printIndent()
		p.printf("Body:\n")
		p.printNodes(n.Body)
	}
}

func (p *Printer) printAssignment(n *ast.AssignmentNode) {
	p.printIndent()
	p.printf("Left:\n")
	p.printNode(n.Left)
	p.printIndent()
	p.printf("Right:\n")
	p.printNode(n.Right)
}

func (p *Printer) printBinaryExpr(n *ast.BinaryExpr) {
	p.printIndent()
	p.printf("Operator: %s\n", n.Operator)
	p.printIndent()
	p.printf("Left:\n")
	p.printNode(n.Left)
	p.printIndent()
	p.printf("Right:\n")
	p.printNode(n.Right)
}

func (p *Printer) printReturn(n *ast.ReturnNode) {
	if n.Expr != nil {
		p.printIndent()
		p.printf("Expression:\n")
		p.printNode(n.Expr)
	}
}

func (p *Printer) printExpressionStmt(n *ast.ExpressionStmt) {
	if n.Expr != nil {
		p.printIndent()
		p.printf("Expression:\n")
		p.printNode(n.Expr)
	}
}

func (p *Printer) printIf(n *ast.IfNode) {
	p.printIndent()
	p.printf("Condition:\n")
	p.printNode(n.Condition)
	if len(n.Body) > 0 {
		p.printIndent()
		p.printf("Then:\n")
		p.printNodes(n.Body)
	}
	for _, elseif := range n.ElseIfs {
		p.printIndent()
		p.printf("ElseIf:\n")
		p.indent++
		p.printIndent()
		p.printf("Condition:\n")
		p.printNode(elseif.Condition)
		if len(elseif.Body) > 0 {
			p.printIndent()
			p.printf("Body:\n")
			p.printNodes(elseif.Body)
		}
		p.indent--
	}
	if n.Else != nil {
		p.printIndent()
		p.printf("Else:\n")
		p.printNodes(n.Else.Body)
	}
}

func (p *Printer) printWhile(n *ast.WhileNode) {
	p.printIndent()
	p.printf("Condition:\n")
	p.printNode(n.Condition)
	if len(n.Body) > 0 {
		p.printIndent()
		p.printf("Body:\n")
		p.printNodes(n.Body)
	}
}

func (p *Printer) printInterpolatedString(n *ast.InterpolatedStringLiteral) {
	p.printIndent()
	p.printf("Parts:\n")
	p.printNodes(n.Parts)
}

func (p *Printer) printClass(n *ast.ClassNode) {
	if n.Extends != "" {
		p.printIndent()
		p.printf("Extends: %s\n", n.Extends)
	}
	if len(n.Implements) > 0 {
		p.printIndent()
		p.printf("Implements: %s\n", strings.Join(n.Implements, ", "))
	}
	if len(n.Properties) > 0 {
		p.printIndent()
		p.printf("Properties:\n")
		p.printNodes(n.Properties)
	}
	if len(n.Methods) > 0 {
		p.printIndent()
		p.printf("Methods:\n")
		p.printNodes(n.Methods)
	}
}

func (p *Printer) printEnum(n *ast.EnumNode) {
	if n.BackedBy != "" {
		p.printIndent()
		p.printf("Backed by: %s\n", n.BackedBy)
	}
	if len(n.Cases) > 0 {
		p.printIndent()
		p.printf("Cases:\n")
		p.printNodes(convertEnumCasesToNodes(n.Cases))
	}
}

func (p *Printer) printVariable(n *ast.VariableNode) {
	p.printIndent()
	p.printf("Name: $%s\n", n.Name)
}

func (p *Printer) printStringLiteral(n *ast.StringLiteral) {
	p.printIndent()
	p.printf("Value: %q\n", n.Value)
}

func (p *Printer) printIntegerLiteral(n *ast.IntegerLiteral) {
	p.printIndent()
	p.printf("Value: %d\n", n.Value)
}

func (p *Printer) printFloatLiteral(n *ast.FloatLiteral) {
	p.printIndent()
	p.printf("Value: %g\n", n.Value)
}

func (p *Printer) printBooleanLiteral(n *ast.BooleanLiteral) {
	p.printIndent()
	p.printf("Value: %t\n", n.Value)
}

func (p *Printer) printNullLiteral(n *ast.NullLiteral) {
	p.printIndent()
	p.printf("Value: null\n")
}

// Helper function to convert enum cases to nodes for printing
func convertEnumCasesToNodes(cases []*ast.EnumCaseNode) []ast.Node {
	nodes := make([]ast.Node, len(cases))
	for i, c := range cases {
		nodes[i] = c
	}
	return nodes
}
