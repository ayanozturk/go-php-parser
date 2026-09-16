# syntax/lower — declaration-tier CST → classic AST

Lowers a `syntax` red tree to classic `[]ast.Node` for project indexing
(`analyse.BuildProjectIndex`) without calling the classic `parser` package.

## Coverage

- File / namespace / use / class / interface / trait / enum
- Methods & functions (signatures); bodies empty when `KindTokenList` or absent
- Full `syntax.Parse` bodies: `KindStatementList` → statement/expression lower
  (ExpressionStmt, Return, If/ElseIf/Else, EmptyStmt; Variable, Literal, Binary,
  Assign, Call/MethodCall, MemberAccess/PropertyFetch, Paren unwrap; stretch
  Unary/Cast/StaticMemberAccess)
- Params, properties (incl. property hooks name/by-ref/parameter header), class constants
- Trait use clauses (incl. adaptations) in class `Properties` / trait `Body`
- Enum cases (names only in index mode; case values nil) and enum methods
- Leading `T_DOC_COMMENT` PHPDoc on class/interface/function/method/property/param/const
  (multi-property shares one doc; enum cases intentionally get no PHPDoc)
- Types: named/primitive, nullable, union, intersection, parenthesized

## Explicit gaps

- No attributes (classic skips them too for index)
- No anonymous class, closures, or arrow functions
- Unsupported stmt/expr kinds still return nil (skip) — loops, try, match, array,
  new, ternary, interpolations, etc. are not fully lowered yet
- Hook Expr/Body and enum case values stay nil in index mode
- TraitNode / EnumNode have no PHPDoc storage field (classic likewise)
- Method/function bodies stay empty in index mode (`ParseASTForIndex`)
- Analyse consumers (`parseAnalysisFile` / semantic_cache) still use classic parse
  by default — body lower unblocks cutover but does not switch it yet
