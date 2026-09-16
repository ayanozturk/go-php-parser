# syntax/lower — declaration-tier CST → classic AST

Lowers a `syntax` red tree to classic `[]ast.Node` for project indexing
(`analyse.BuildProjectIndex`) without calling the classic `parser` package.

## MVP coverage

- File / namespace / use / class / interface
- Methods & functions (signatures); bodies empty when `KindTokenList` or absent
- Params, properties, class constants (name ± type)
- Types: named/primitive, nullable, union, intersection, parenthesized

## Explicit gaps

- No PHPDoc / attributes / property hooks
- No enum, trait, anonymous class, closures, or arrow functions
- No full expression or statement lowering
- Method/function bodies stay empty in index mode (`ParseASTForIndex`)
