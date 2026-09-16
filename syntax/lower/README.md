# syntax/lower — declaration-tier CST → classic AST

Lowers a `syntax` red tree to classic `[]ast.Node` for project indexing
(`analyse.BuildProjectIndex`) without calling the classic `parser` package.

## Coverage

- File / namespace / use / class / interface / trait / enum
- Methods & functions (signatures); bodies empty when `KindTokenList` or absent
- Full `syntax.Parse` bodies: `KindStatementList` → statement/expression lower
  - Stmts: ExpressionStmt (incl. file-scope), Return, If/ElseIf/Else, EmptyStmt, Echo,
    Throw, While/DoWhile/For/Foreach, Break/Continue, Try/Catch/Finally, Switch/Case/Default
  - Exprs: Variable, Literal, Binary, Assign, Ternary, ArrayAccess, Array/ArrayElement/List,
    New (named/`$var`/dynamic/anonymous class), Call/MethodCall/ArgList,
    MemberAccess/PropertyFetch (incl. incomplete `$obj->`), StaticMemberAccess,
    Closure/Arrow/Match/Clone, Include/Print, InterpolatedString/Heredoc/Nowdoc,
    Cast (incl. `(unset)`), ThrowExpr, Paren unwrap, Unary
- Params, properties (incl. property hooks name/by-ref/parameter header), class constants
- Trait use clauses (incl. adaptations) in class `Properties` / trait `Body`
- Enum cases (names only in index mode; case values nil) and enum methods
- Leading `T_DOC_COMMENT` PHPDoc on class/interface/function/method/property/param/const
  (multi-property shares one doc; enum cases intentionally get no PHPDoc)
- Types: named/primitive, nullable, union, intersection, parenthesized

## Explicit gaps

- No attributes (classic skips them too for index)
- Yield, first-class callable, variable-variables still skip (nil)
- Hook Expr/Body stay nil in index mode; const/enum case **values** lower in full ParseAST
- TraitNode / EnumNode have no PHPDoc storage field (classic likewise)
- Method/function bodies stay empty in index mode (`ParseASTForIndex`)
- Analyse consumers use `syntax.ParseAST` (`command.parseAnalysisFile` / Strom semantic_cache)
