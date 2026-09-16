# syntax/lower — declaration-tier CST → classic AST

Lowers a `syntax` red tree to classic `[]ast.Node` for project indexing
(`analyse.BuildProjectIndex`) without calling the classic `parser` package.

## Coverage

- File / namespace / use / class / interface / trait / enum
- Methods & functions (signatures); bodies empty when `KindTokenList` or absent
- Full `syntax.Parse` bodies: `KindStatementList` → statement/expression lower
  - Stmts: ExpressionStmt, Return, If/ElseIf/Else, EmptyStmt, Echo (→ ExpressionStmt/Block),
    Throw, While/DoWhile/For/Foreach, Break/Continue, Try/Catch/Finally, Switch/Case/Default
  - Exprs: Variable, Literal, Binary, Assign, Ternary (incl. Elvis), ArrayAccess, Array/ArrayElement,
    New (named/`$var`/dynamic; anonymous class skipped), Call/MethodCall/ArgList,
    MemberAccess/PropertyFetch, StaticMemberAccess → ClassConstFetch,
    `Class::method()` Call → FunctionCall `Class::method`, Paren unwrap, Unary, Cast, ThrowExpr
- Params, properties (incl. property hooks name/by-ref/parameter header), class constants
- Trait use clauses (incl. adaptations) in class `Properties` / trait `Body`
- Enum cases (names only in index mode; case values nil) and enum methods
- Leading `T_DOC_COMMENT` PHPDoc on class/interface/function/method/property/param/const
  (multi-property shares one doc; enum cases intentionally get no PHPDoc)
- Types: named/primitive, nullable, union, intersection, parenthesized

## Explicit gaps

- No attributes (classic skips them too for index)
- No anonymous class, closures, or arrow functions
- Match, heredoc/nowdoc, encapsed interpolations, list-destructure, yield/clone/first-class
  callable, include/print, variable-variables still skip (nil)
- Hook Expr/Body and enum case values stay nil in index mode
- TraitNode / EnumNode have no PHPDoc storage field (classic likewise)
- Method/function bodies stay empty in index mode (`ParseASTForIndex`)
- Analyse consumers use `syntax.ParseAST` (`command.parseAnalysisFile`) / Strom
  `parseSemanticSnapshot` after action 98 cutover
