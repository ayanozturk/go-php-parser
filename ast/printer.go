package ast

import "fmt"

// PrintAST recursively prints the AST tree with indentation
func PrintAST(nodes []Node, indent int) {
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
		case *FunctionNode:
			if len(n.Params) > 0 {
				fmt.Println(prefix + "  Parameters:")
				PrintAST(n.Params, indent+2)
			}
			if len(n.Body) > 0 {
				fmt.Println(prefix + "  Body:")
				PrintAST(n.Body, indent+2)
			}
		case *AssignmentNode:
			fmt.Println(prefix + "  Left:")
			PrintAST([]Node{n.Left}, indent+2)
			fmt.Println(prefix + "  Right:")
			PrintAST([]Node{n.Right}, indent+2)
		case *BinaryExpr:
			fmt.Println(prefix + "  Left:")
			PrintAST([]Node{n.Left}, indent+2)
			fmt.Println(prefix + "  Right:")
			PrintAST([]Node{n.Right}, indent+2)
		case *ReturnNode:
			if n.Expr != nil {
				fmt.Println(prefix + "  Expression:")
				PrintAST([]Node{n.Expr}, indent+2)
			}
		case *ExpressionStmt:
			if n.Expr != nil {
				fmt.Println(prefix + "  Expression:")
				PrintAST([]Node{n.Expr}, indent+2)
			}
		case *IfNode:
			fmt.Println(prefix + "  Condition:")
			PrintAST([]Node{n.Condition}, indent+2)
			if len(n.Body) > 0 {
				fmt.Println(prefix + "  Then:")
				PrintAST(n.Body, indent+2)
			}
			for _, elseif := range n.ElseIfs {
				fmt.Println(prefix + "  ElseIf:")
				fmt.Println(prefix + "    Condition:")
				PrintAST([]Node{elseif.Condition}, indent+3)
				if len(elseif.Body) > 0 {
					fmt.Println(prefix + "    Body:")
					PrintAST(elseif.Body, indent+3)
				}
			}
			if n.Else != nil {
				fmt.Println(prefix + "  Else:")
				PrintAST(n.Else.Body, indent+2)
			}
		case *WhileNode:
			fmt.Println(prefix + "  Condition:")
			PrintAST([]Node{n.Condition}, indent+2)
			if len(n.Body) > 0 {
				fmt.Println(prefix + "  Body:")
				PrintAST(n.Body, indent+2)
			}
		case *InterpolatedStringLiteral:
			fmt.Println(prefix + "  Parts:")
			PrintAST(n.Parts, indent+2)
		case *ClassNode:
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
