package analyse

import (
	"fmt"

	"github.com/ayanozturk/go-php-parser/ast"
)

func checkUndefinedVariables(filename string, nodes []ast.Node, ctx *AnalysisContext) []AnalysisIssue {
	issues := make([]AnalysisIssue, 0)
	forEachVariableRead(filename, nodes, ctx, func(read VariableReadFact) {
		if read.State == VariableDefinitelyDefined {
			return
		}
		issues = append(issues, variableReadIssue(filename, read, level1VariablesCode, fmt.Sprintf("Variable $%s might not be defined.", read.Key.Name)))
	})
	return issues
}

type variableFlowRanger interface {
	rangeVariableReadsForFile(filename string, visit func(VariableReadFact))
}

func forEachVariableRead(filename string, nodes []ast.Node, ctx *AnalysisContext, visit func(VariableReadFact)) {
	if ctx != nil && ctx.VariableFlow != nil {
		if ranger, ok := ctx.VariableFlow.(variableFlowRanger); ok {
			ranger.rangeVariableReadsForFile(filename, visit)
			return
		}
		for _, read := range ctx.VariableFlow.VariableReadsForFile(filename) {
			visit(read)
		}
		return
	}
	reads := buildVariableFlowFacts(filename, nodes)
	if ctx != nil {
		ctx.VariableFlow = singleFileVariableFlow{filename: filename, reads: reads}
	}
	for _, read := range reads {
		visit(read.public(filename))
	}
}

type singleFileVariableFlow struct {
	filename string
	reads    []variableReadFact
}

func (f singleFileVariableFlow) VariableReadsForFile(filename string) []VariableReadFact {
	if filename != f.filename {
		return nil
	}
	result := make([]VariableReadFact, len(f.reads))
	for i, read := range f.reads {
		result[i] = read.public(filename)
	}
	return result
}

func (f singleFileVariableFlow) rangeVariableReadsForFile(filename string, visit func(VariableReadFact)) {
	if filename != f.filename {
		return
	}
	for _, read := range f.reads {
		visit(read.public(filename))
	}
}

func variableReadIssue(filename string, read VariableReadFact, code, message string) AnalysisIssue {
	return AnalysisIssue{
		Filename:  filename,
		Line:      read.Start.Line,
		Column:    read.Start.Column,
		EndLine:   read.End.Line,
		EndColumn: read.End.Column,
		Code:      code,
		Message:   message,
	}
}
