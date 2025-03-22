package printer

import (
	"fmt"
	"go-phpcs/ast"
	"strings"
)

// PrintAST recursively prints the AST tree with indentation
func PrintAST(nodes []ast.Node, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	for _, node := range nodes {
		if node == nil {
			continue
		}
		fmt.Println(prefix + node.String())
		switch n := node.(type) {
		case *ast.ArrayNode:
			if len(n.Elements) > 0 {
				elements := make([]string, len(n.Elements))
				for i, elem := range n.Elements {
					if item, ok := elem.(*ast.ArrayItemNode); ok {
						var itemStr string
						if item.Key != nil {
							itemStr = fmt.Sprintf("%s => %s", item.Key.TokenLiteral(), item.Value.TokenLiteral())
						} else {
							itemStr = item.Value.TokenLiteral()
						}
						if item.ByRef {
							itemStr = "&" + itemStr
						}
						if item.Unpack {
							itemStr = "..." + itemStr
						}
						elements[i] = itemStr
					}
				}
				fmt.Printf("%s  [ %s ]\n", prefix, strings.Join(elements, ", "))
			} else {
				fmt.Printf("%s  []\n", prefix)
			}
		case *ast.ArrayItemNode:
			if n.Key != nil {
				fmt.Println(prefix + "  Key:")
				PrintAST([]ast.Node{n.Key}, indent+2)
			}
			fmt.Println(prefix + "  Value:")
			PrintAST([]ast.Node{n.Value}, indent+2)
			if n.ByRef {
				fmt.Println(prefix + "  ByRef: true")
			}
			if n.Unpack {
				fmt.Println(prefix + "  Unpack: true")
			}
		case *ast.FunctionNode:
			if len(n.Params) > 0 {
				fmt.Println(prefix + "  Parameters:")
				PrintAST(n.Params, indent+2)
			}
			if len(n.Body) > 0 {
				fmt.Println(prefix + "  Body:")
				PrintAST(n.Body, indent+2)
			}
		case *ast.AssignmentNode:
			fmt.Println(prefix + "  Left:")
			PrintAST([]ast.Node{n.Left}, indent+2)
			fmt.Println(prefix + "  Right:")
			PrintAST([]ast.Node{n.Right}, indent+2)
		case *ast.BinaryExpr:
			fmt.Println(prefix + "  Left:")
			PrintAST([]ast.Node{n.Left}, indent+2)
			fmt.Println(prefix + "  Right:")
			PrintAST([]ast.Node{n.Right}, indent+2)
		case *ast.ReturnNode:
			if n.Expr != nil {
				fmt.Println(prefix + "  Expression:")
				PrintAST([]ast.Node{n.Expr}, indent+2)
			}
		case *ast.ExpressionStmt:
			if n.Expr != nil {
				fmt.Println(prefix + "  Expression:")
				PrintAST([]ast.Node{n.Expr}, indent+2)
			}
		case *ast.IfNode:
			fmt.Println(prefix + "  Condition:")
			PrintAST([]ast.Node{n.Condition}, indent+2)
			if len(n.Body) > 0 {
				fmt.Println(prefix + "  Then:")
				PrintAST(n.Body, indent+2)
			}
			for _, elseif := range n.ElseIfs {
				fmt.Println(prefix + "  ElseIf:")
				fmt.Println(prefix + "    Condition:")
				PrintAST([]ast.Node{elseif.Condition}, indent+3)
				if len(elseif.Body) > 0 {
					fmt.Println(prefix + "    Body:")
					PrintAST(elseif.Body, indent+3)
				}
			}
			if n.Else != nil {
				fmt.Println(prefix + "  Else:")
				PrintAST(n.Else.Body, indent+2)
			}
		case *ast.WhileNode:
			fmt.Println(prefix + "  Condition:")
			PrintAST([]ast.Node{n.Condition}, indent+2)
			if len(n.Body) > 0 {
				fmt.Println(prefix + "  Body:")
				PrintAST(n.Body, indent+2)
			}
		case *ast.InterpolatedStringLiteral:
			fmt.Println(prefix + "  Parts:")
			PrintAST(n.Parts, indent+2)
		case *ast.ClassNode:
			if len(n.Properties) > 0 {
				fmt.Println(prefix + "  Properties:")
				PrintAST(n.Properties, indent+2)
			}
			if len(n.Methods) > 0 {
				fmt.Println(prefix + "  Methods:")
				PrintAST(n.Methods, indent+2)
			}
		}
	}
}
