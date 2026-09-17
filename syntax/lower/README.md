# syntax/lower — declaration-tier CST → classic AST

Lowers a `syntax` red tree to classic `[]ast.Node` for project indexing
(`analyse.BuildProjectIndex`) without calling the classic `parser` package.

This is a transitional bridge, not a permanent layer: `analyse/` rules are
being migrated one at a time to walk the CST directly (via `syntax.Walk` +
`RedNode.Pos()`/`EndPos()`) and skip lowering entirely. See `AGENTS.md`
"CST-direct migration" for status; `analyse/empty_statement_rule.go` is the
first rule migrated off this path when raw source is available.

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
- Params, properties (incl. property hooks name/by-ref/parameter/Expr/Body, and
  per-name `DefaultValue`), class constants
- Member / param / method attributes (`Attributes` on Function/Property/Param/Const)
- Trait use clauses (incl. adaptations) in class `Properties` / trait `Body`
- Enum cases (names + values when present) and enum methods
- Leading `T_DOC_COMMENT` PHPDoc on class/interface/function/method/property/param/const
  (multi-property shares one doc; enum cases intentionally get no PHPDoc)
- Types: named/primitive, nullable, union, intersection, parenthesized
- Braced string interpolations `{$expr}` / `${name}` in Parts; dynamic static
  members `Foo::{$m}` → `ClassConstFetchNode.ConstExpr`

## Explicit gaps

- Hook Expr/Body lower when CST has arrow expr / `KindStatementList` (hooks are
  never body-blobbed by `SkipFunctionBodies`; abstract `get;` / `set;` stay nil)
- TraitNode / EnumNode have no PHPDoc storage field (classic likewise)
- Method/function bodies stay empty in index mode (`ParseASTForIndex`)
- Analyse consumers use `syntax.ParseAST` (`command.parseAnalysisFile` / Strom semantic_cache)
