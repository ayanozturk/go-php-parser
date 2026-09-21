package analyse

import (
	"fmt"

	"github.com/ayanozturk/go-php-parser/syntax"
)

// appendPropertyCallableTypeIssueFromCST is the CST-native analogue of
// appendPropertyCallableTypeIssue for KindPropertyDecl: reads the shared type
// text + variable names without LowerPropertyDeclNode.
//
// Span parity with the Lower* path: single-name decls use the whole property
// decl span; multi-name decls use each variable token span.
func appendPropertyCallableTypeIssueFromCST(filename string, propDecl *syntax.RedNode, issues *[]AnalysisIssue) {
	if propDecl == nil || propDecl.Kind() != syntax.KindPropertyDecl {
		return
	}
	typ := syntax.PropertyDeclType(propDecl)
	if typ == nil {
		return
	}
	raw := syntax.TypeText(typ)
	if !typeContainsNativeCallable(raw) {
		return
	}
	vars := syntax.PropertyDeclVariables(propDecl)
	if len(vars) == 0 {
		return
	}
	if len(vars) == 1 {
		name := syntax.VariableName(vars[0])
		*issues = append(*issues, issueSpanRed(filename, propDecl, level0PropertyCallableTypeCode,
			fmt.Sprintf("Property $%s cannot have callable in its type declaration.", name)))
		return
	}
	for _, v := range vars {
		name := syntax.VariableName(v)
		start, end := syntax.RawSpanPositions(v)
		*issues = append(*issues, AnalysisIssue{
			Filename:  filename,
			Line:      start.Line,
			Column:    start.Column,
			EndLine:   end.Line,
			EndColumn: end.Column,
			Code:      level0PropertyCallableTypeCode,
			Message:   fmt.Sprintf("Property $%s cannot have callable in its type declaration.", name),
		})
	}
}

// appendPromotedParamCallableTypeIssueFromCST is the CST-native analogue of
// appendPropertyCallableTypeIssue for promoted KindParam (constructor
// property promotion). Non-promoted params are ignored, matching the AST path.
func appendPromotedParamCallableTypeIssueFromCST(filename string, param *syntax.RedNode, issues *[]AnalysisIssue) {
	if param == nil || param.Kind() != syntax.KindParam {
		return
	}
	if !syntax.ParamIsPromoted(param) {
		return
	}
	typ := syntax.ParamType(param)
	if typ == nil {
		return
	}
	raw := syntax.TypeText(typ)
	if !typeContainsNativeCallable(raw) {
		return
	}
	name := syntax.VariableName(syntax.ParamVariable(param))
	*issues = append(*issues, issueSpanRed(filename, param, level0PropertyCallableTypeCode,
		fmt.Sprintf("Property $%s cannot have callable in its type declaration.", name)))
}
