package syntax

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/token"
)

func lowerNamespace(n *RedNode, file *File, siblings []*RedNode, idx int) (*ast.NamespaceNode, int) {
	pos, end := nodePos(file, n)
	ns := &ast.NamespaceNode{Pos: pos, EndPos: end}
	var bodyList *RedNode
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if isNameKind(k) {
			nameNode := RedNode{File: n.File, Green: green, Offset: offset}
			ns.Name = nameString(&nameNode)
		}
		if k == KindStatementList {
			bodyList = &RedNode{File: n.File, Green: green, Offset: offset}
		}
		return true
	})
	if bodyList != nil {
		ns.Body = lowerStatementChildren(bodyList, file)
		return ns, 0
	}
	// Inline `namespace Name;` — fold following top-level decls into Body so
	// index consumers see a single NamespaceNode (matches MVP fixture asserts).
	consumed := 0
	var body []ast.Node
	for j := idx + 1; j < len(siblings); j++ {
		sib := siblings[j]
		if sib.Kind() == KindNamespaceDecl {
			break
		}
		consumed++
		nodes, _ := lowerTopLevel(sib, file)
		body = append(body, nodes...)
	}
	ns.Body = body
	return ns, consumed
}

func lowerStatementChildren(list *RedNode, file *File) []ast.Node {
	if list == nil {
		return nil
	}
	var out []ast.Node
	list.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindToken {
			return true
		}
		child := RedNode{File: list.File, Green: green, Offset: offset}
		nodes, _ := lowerTopLevel(&child, file)
		out = append(out, nodes...)
		return true
	})
	return out
}

func lowerUseDecl(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	useType := "class"
	n.ForEachChildDesc(func(green *GreenNode, _ int) bool {
		if green.Kind() != KindToken {
			return true
		}
		switch green.TokenType() {
		case token.T_FUNCTION:
			useType = "function"
		case token.T_CONST:
			useType = "const"
		}
		return true
	})
	var out []ast.Node
	for _, clause := range n.ChildrenOfKind(KindUseClause) {
		out = append(out, lowerUseClause(clause, "", useType, pos, end)...)
	}
	return out
}

func lowerUseClause(clause *RedNode, prefix, useType string, pos, end ast.Position) []ast.Node {
	if clause == nil {
		return nil
	}
	itemType := useType
	var name *RedNode
	var alias *RedNode
	var group *RedNode
	clause.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch {
		case green.Kind() == KindToken && green.TokenType() == token.T_FUNCTION:
			itemType = "function"
		case green.Kind() == KindToken && green.TokenType() == token.T_CONST:
			itemType = "const"
		case isNameKind(green.Kind()):
			c := clause.bindChild(green, offset)
			if name == nil {
				name = c
			} else {
				alias = c
			}
		case green.Kind() == KindUseGroup:
			group = clause.bindChild(green, offset)
		}
		return true
	})
	if group != nil {
		base := strings.TrimPrefix(NameText(name), `\`)
		if prefix != "" {
			base = strings.Trim(prefix, `\`) + `\` + strings.Trim(base, `\`)
		}
		base = strings.TrimSuffix(base, `\`)
		var out []ast.Node
		for _, inner := range group.ChildrenOfKind(KindUseClause) {
			out = append(out, lowerUseClause(inner, base, itemType, pos, end)...)
		}
		return out
	}
	path := strings.TrimPrefix(NameText(name), `\`)
	if prefix != "" {
		path = strings.Trim(prefix, `\`) + `\` + path
	}
	path = strings.Trim(path, `\`)
	aliasName := ""
	if alias != nil {
		aliasName = NameText(alias)
	} else {
		aliasName = unqualifiedTail(path)
	}
	return []ast.Node{&ast.UseNode{
		Path:   path,
		Alias:  aliasName,
		Type:   itemType,
		Pos:    pos,
		EndPos: end,
	}}
}

func lowerClass(n *RedNode, file *File) *ast.ClassNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	cls := &ast.ClassNode{
		Pos:       pos,
		EndPos:    end,
		Modifiers: lowerModifiers(n),
		PHPDoc:    leadingDocFromNode(n),
	}
	var members *RedNode
	headerEnd := pos
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch green.Kind() {
		case KindUnqualifiedName, KindQualifiedName,
			KindFullyQualifiedName, KindRelativeName:
			if cls.Name == "" {
				nameNode := RedNode{File: n.File, Green: green, Offset: offset}
				cls.Name = unqualifiedTail(NameText(&nameNode))
				headerEnd = spanEnd(file, nameNode.Span())
			}
		case KindExtendsClause:
			clause := RedNode{File: n.File, Green: green, Offset: offset}
			names := clauseNames(&clause)
			if len(names) > 0 {
				cls.Extends = names[0]
			}
			headerEnd = spanEnd(file, clause.Span())
		case KindImplementsClause:
			clause := RedNode{File: n.File, Green: green, Offset: offset}
			cls.Implements = clauseNames(&clause)
			headerEnd = spanEnd(file, clause.Span())
		case KindMemberList:
			members = &RedNode{File: n.File, Green: green, Offset: offset}
		case KindModifierList:
			modList := RedNode{File: n.File, Green: green, Offset: offset}
			headerEnd = spanEnd(file, modList.Span())
		}
		return true
	})
	cls.HeaderEndPos = headerEnd
	if members != nil {
		lowerClassMembers(members, file, cls)
	}
	return cls
}

func lowerInterface(n *RedNode, file *File) *ast.InterfaceNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	iface := &ast.InterfaceNode{
		Pos:    pos,
		EndPos: end,
		PHPDoc: leadingDocFromNode(n),
	}
	var members *RedNode
	headerEnd := pos
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		switch green.Kind() {
		case KindUnqualifiedName, KindQualifiedName,
			KindFullyQualifiedName, KindRelativeName:
			if iface.Name == "" {
				nameNode := RedNode{File: n.File, Green: green, Offset: offset}
				iface.Name = unqualifiedTail(NameText(&nameNode))
				headerEnd = spanEnd(file, nameNode.Span())
			}
		case KindExtendsClause:
			clause := RedNode{File: n.File, Green: green, Offset: offset}
			iface.Extends = clauseNames(&clause)
			headerEnd = spanEnd(file, clause.Span())
		case KindMemberList:
			members = &RedNode{File: n.File, Green: green, Offset: offset}
		}
		return true
	})
	iface.HeaderEndPos = headerEnd
	if members != nil {
		var pendingAttrs []ast.Node
		memberKinds := func(k Kind) bool {
			switch k {
			case KindAttributeList, KindFunctionDecl, KindMethodDecl, KindClassConstDecl, KindPropertyDecl:
				return true
			default:
				return false
			}
		}
		members.ForEachChildDesc(func(green *GreenNode, offset int) bool {
			k := green.Kind()
			if !memberKinds(k) {
				pendingAttrs = nil
				return true
			}
			m := &RedNode{File: members.File, Green: green, Offset: offset}
			switch k {
			case KindAttributeList:
				pendingAttrs = append(pendingAttrs, lowerAttributeList(m, file)...)
			case KindFunctionDecl, KindMethodDecl:
				if im := lowerInterfaceMethod(m, file); im != nil {
					if len(pendingAttrs) > 0 {
						im.Attributes = pendingAttrs
						pendingAttrs = nil
					}
					iface.Members = append(iface.Members, im)
				}
			case KindClassConstDecl:
				consts := lowerClassConsts(m, file)
				if len(pendingAttrs) > 0 {
					for _, node := range consts {
						if c, ok := node.(*ast.ConstantNode); ok {
							c.Attributes = pendingAttrs
						}
					}
					pendingAttrs = nil
				}
				iface.Members = append(iface.Members, consts...)
			case KindPropertyDecl:
				props := lowerProperties(m, file)
				if len(pendingAttrs) > 0 {
					for _, node := range props {
						if prop, ok := node.(*ast.PropertyNode); ok {
							prop.Attributes = pendingAttrs
						}
					}
					pendingAttrs = nil
				}
				// PHP 8.4 interface property hooks.
				iface.Members = append(iface.Members, props...)
			default:
				pendingAttrs = nil
			}
			return true
		})
	}
	return iface
}

func lowerClassMembers(members *RedNode, file *File, cls *ast.ClassNode) {
	var traitUses []ast.Node
	var properties []ast.Node
	var pendingAttrs []ast.Node
	memberKinds := func(k Kind) bool {
		switch k {
		case KindAttributeList, KindFunctionDecl, KindMethodDecl, KindPropertyDecl, KindClassConstDecl, KindUseTraitClause:
			return true
		default:
			return false
		}
	}
	members.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		if !memberKinds(k) {
			pendingAttrs = nil
			return true
		}
		m := &RedNode{File: members.File, Green: green, Offset: offset}
		switch k {
		case KindAttributeList:
			pendingAttrs = append(pendingAttrs, lowerAttributeList(m, file)...)
		case KindFunctionDecl, KindMethodDecl:
			if fn := lowerFunction(m, file); fn != nil {
				if len(pendingAttrs) > 0 {
					fn.Attributes = pendingAttrs
					pendingAttrs = nil
				}
				cls.Methods = append(cls.Methods, fn)
			}
		case KindPropertyDecl:
			props := lowerProperties(m, file)
			if len(pendingAttrs) > 0 {
				for _, node := range props {
					if prop, ok := node.(*ast.PropertyNode); ok {
						prop.Attributes = pendingAttrs
					}
				}
				pendingAttrs = nil
			}
			properties = append(properties, props...)
		case KindClassConstDecl:
			consts := lowerClassConsts(m, file)
			if len(pendingAttrs) > 0 {
				for _, node := range consts {
					if c, ok := node.(*ast.ConstantNode); ok {
						c.Attributes = pendingAttrs
					}
				}
				pendingAttrs = nil
			}
			cls.Constants = append(cls.Constants, consts...)
		case KindUseTraitClause:
			pendingAttrs = nil // attributes do not attach to `use` clauses
			if tu := lowerUseTraitClause(m, file); tu != nil {
				traitUses = append(traitUses, tu)
			}
		}
		return true
	})
	// Classic prepends trait uses ahead of properties in ClassNode.Properties.
	if len(traitUses) > 0 {
		cls.Properties = append(traitUses, properties...)
	} else {
		cls.Properties = properties
	}
}

func lowerFunction(n *RedNode, file *File) *ast.FunctionNode {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	// Classic anchors Pos at T_FUNCTION (modifiers are outside the Pos span).
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() == KindToken && green.TokenType() == token.T_FUNCTION {
			pos, _ = nodePosGreen(file, green, offset)
			return false
		}
		return true
	})
	fn := &ast.FunctionNode{
		Pos:       pos,
		EndPos:    end,
		Modifiers: lowerModifiers(n),
		PHPDoc:    leadingDocFromNode(n),
	}
	seenColon := false
	headerEnd := pos
	n.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		k := green.Kind()
		switch k {
		case KindUnqualifiedName:
			if fn.Name == "" {
				nameNode := RedNode{File: n.File, Green: green, Offset: offset}
				fn.Name = NameText(&nameNode)
				headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
			}
		case KindParamList:
			paramList := RedNode{File: n.File, Green: green, Offset: offset}
			fn.Params = lowerParamList(&paramList, file)
			headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
		case KindTokenList:
			// Index mode: leave Body nil.
			fn.Body = nil
		case KindStatementList:
			bodyList := RedNode{File: n.File, Green: green, Offset: offset}
			fn.Body = lowerStatements(&bodyList, file)
		default:
			if k == KindToken && green.TokenType() == token.T_COLON {
				seenColon = true
				headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
				return true
			}
			if seenColon && isTypeKind(k) {
				typeNode := RedNode{File: n.File, Green: green, Offset: offset}
				fn.ReturnType = lowerType(&typeNode, file)
				headerEnd = spanEnd(file, Span{Start: offset, End: offset + green.width})
				seenColon = false
			}
		}
		return true
	})
	fn.HeaderEndPos = headerEnd
	return fn
}

func lowerInterfaceMethod(n *RedNode, file *File) *ast.InterfaceMethodNode {
	fn := lowerFunction(n, file)
	if fn == nil {
		return nil
	}
	return &ast.InterfaceMethodNode{
		Name:       fn.Name,
		Modifiers:  fn.Modifiers,
		ReturnType: fn.ReturnType,
		Params:     fn.Params,
		Attributes: fn.Attributes,
		PHPDoc:     fn.PHPDoc,
		Pos:        fn.Pos,
		EndPos:     fn.EndPos,
	}
}

func lowerProperties(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	pos, end := nodePos(file, n)
	mods := lowerModifiers(n)
	phpdoc := leadingDocFromNode(n)
	var typeHint ast.Node
	var hooks []ast.PropertyHookNode
	type propName struct {
		name         string
		pos          ast.Position
		end          ast.Position
		defaultValue ast.Node
	}
	var names []propName
	i := 0
	for {
		prev := i
		i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
			k := green.Kind()
			switch {
			case isTypeKind(k):
				typeNode := RedNode{File: n.File, Green: green, Offset: offset}
				typeHint = lowerType(&typeNode, file)
				return -1
			case isGreenTokenType(green, token.T_VARIABLE):
				sp := Span{Start: offset, End: offset + green.width}
				nm := propName{
					name: stripVarDollar(greenTokenLiteral(green)),
					pos:  spanStart(file, sp),
					end:  spanEnd(file, sp),
				}
				// Optional `= <expr>` default immediately after the variable.
				j := idx + 1
				hasAssign := false
				walkRedNodeChildrenFrom(n, j, func(peek *GreenNode, peekOff int, pj int) int {
					if pj != j {
						return pj
					}
					hasAssign = isGreenTokenType(peek, token.T_ASSIGN)
					return pj
				})
				if hasAssign {
					j = walkRedNodeChildrenFrom(n, j+1, func(v *GreenNode, vOff int, vj int) int {
						if isGreenTokenType(v, token.T_COMMA) || isGreenTokenType(v, token.T_SEMICOLON) ||
							v.Kind() == KindPropertyHookList {
							return vj
						}
						vk := v.Kind()
						if isExprKind(vk) || isNameKind(vk) {
							nm.defaultValue = lowerExprAt(file, v, vOff)
							if nm.defaultValue != nil {
								nm.end = nm.defaultValue.GetEndPos()
							}
							return vj + 1
						}
						return -1
					})
				}
				names = append(names, nm)
				return j
			case k == KindPropertyHookList:
				hookList := RedNode{File: n.File, Green: green, Offset: offset}
				hooks = lowerPropertyHooks(&hookList, file)
				return -1
			}
			return -1
		})
		if i <= prev {
			break
		}
	}
	if len(names) == 0 {
		return nil
	}
	out := make([]ast.Node, 0, len(names))
	for _, nm := range names {
		prop := &ast.PropertyNode{
			Name:         nm.name,
			TypeHint:     typeHint,
			PHPDoc:       phpdoc, // multi-property: same doc on all names (classic)
			DefaultValue: nm.defaultValue,
			Modifiers:    mods,
			IsStatic:     mods.HasName("static"),
			IsReadonly:   mods.HasName("readonly"),
			Hooks:        hooks,
			Pos:          pos,
			EndPos:       end,
		}
		if len(names) == 1 {
			prop.Pos = pos
			prop.EndPos = end
		} else {
			prop.Pos = nm.pos
			prop.EndPos = nm.end
		}
		out = append(out, prop)
	}
	return out
}

func lowerPropertyHooks(list *RedNode, file *File) []ast.PropertyHookNode {
	if list == nil {
		return nil
	}
	var out []ast.PropertyHookNode
	list.ForEachChildDesc(func(green *GreenNode, offset int) bool {
		if green.Kind() != KindPropertyHook {
			return true
		}
		hook := RedNode{File: list.File, Green: green, Offset: offset}
		if h, ok := lowerPropertyHook(&hook, file); ok {
			out = append(out, h)
		}
		return true
	})
	return out
}

func lowerPropertyHook(n *RedNode, file *File) (ast.PropertyHookNode, bool) {
	pos, end := nodePos(file, n)
	h := ast.PropertyHookNode{Pos: pos, EndPos: end}
	i := 0
	for {
		prev := i
		i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
			if isGreenTokenType(green, token.T_AMPERSAND) {
				h.IsByRef = true
				return -1
			}
			if isGreenTokenType(green, token.T_STRING) && h.Name == "" {
				h.Name = greenTokenLiteral(green)
				return -1
			}
			if isGreenTokenType(green, token.T_LPAREN) {
				// Capture balanced header text like classic readBalancedPropertyHookHeader.
				start := offset
				next := walkRedNodeChildrenFrom(n, idx, func(inner *GreenNode, innerOff int, j int) int {
					if isGreenTokenType(inner, token.T_RPAREN) {
						endOff := innerOff + inner.width
						if file != nil && start >= 0 && endOff <= len(file.Source) && start <= endOff {
							h.Parameter = string(file.Source[start:endOff])
						}
						return j + 1
					}
					return -1
				})
				return next
			}
			// Arrow hook: get => <expr>;  /  set($v) => <expr>;
			if isGreenTokenType(green, token.T_DOUBLE_ARROW) {
				next := walkRedNodeChildrenFrom(n, idx+1, func(v *GreenNode, vOff int, j int) int {
					if isGreenTokenType(v, token.T_SEMICOLON) {
						return j
					}
					vk := v.Kind()
					if isExprKind(vk) || isNameKind(vk) {
						h.Expr = lowerExprAt(file, v, vOff)
						return j + 1
					}
					return -1
				})
				return next
			}
			// Braced hook body: get { … } / set($v) { … }
			if green.Kind() == KindStatementList {
				body := RedNode{File: n.File, Green: green, Offset: offset}
				h.Body = lowerStatements(&body, file)
				return -1
			}
			// Abstract / interface: bare `get;` / `set;` — Expr and Body stay nil.
			return -1
		})
		if i <= prev {
			break
		}
	}
	if h.Name == "" {
		return h, false
	}
	return h, true
}

func lowerClassConsts(n *RedNode, file *File) []ast.Node {
	if n == nil {
		return nil
	}
	_, end := nodePos(file, n)
	mods := lowerModifiers(n)
	phpdoc := leadingDocFromNode(n)
	var typeHint ast.Node
	var out []ast.Node
	i := 0
	for {
		prev := i
		i = walkRedNodeChildrenFrom(n, i, func(green *GreenNode, offset int, idx int) int {
			k := green.Kind()
			if isTypeKind(k) {
				typeNode := RedNode{File: n.File, Green: green, Offset: offset}
				typeHint = lowerType(&typeNode, file)
				return -1
			}
			if k != KindUnqualifiedName {
				return -1
			}
			nameNode := RedNode{File: n.File, Green: green, Offset: offset}
			name := NameText(&nameNode)
			np, ne := nodePosGreen(file, green, offset)
			var value ast.Node
			j := idx + 1
			hasAssign := false
			walkRedNodeChildrenFrom(n, j, func(peek *GreenNode, peekOff int, pj int) int {
				if pj != j {
					return pj
				}
				hasAssign = isGreenTokenType(peek, token.T_ASSIGN)
				return pj
			})
			if hasAssign {
				j = walkRedNodeChildrenFrom(n, j+1, func(v *GreenNode, vOff int, vj int) int {
					if isGreenTokenType(v, token.T_COMMA) || isGreenTokenType(v, token.T_SEMICOLON) {
						return vj
					}
					vk := v.Kind()
					if isExprKind(vk) || isNameKind(vk) {
						value = lowerExprAt(file, v, vOff)
						if value != nil {
							ne = value.GetEndPos()
						}
						return vj + 1
					}
					return -1
				})
			}
			out = append(out, &ast.ConstantNode{
				Name:      name,
				Value:     value,
				Type:      typeHint,
				PHPDoc:    phpdoc,
				Modifiers: mods,
				Pos:       np,
				EndPos:    end,
			})
			_ = ne
			return j
		})
		if i <= prev {
			break
		}
	}
	return out
}
