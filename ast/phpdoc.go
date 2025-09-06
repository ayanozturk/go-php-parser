package ast

import (
	"fmt"
	"strings"
)

// PHPDocNode represents a parsed PHPDoc block
type PHPDocNode struct {
	RawContent  string
	Params      []PHPDocParam
	ReturnType  string
	VarType     string
	Description string
	Pos         Position
}

func (p *PHPDocNode) NodeType() string    { return "PHPDoc" }
func (p *PHPDocNode) GetPos() Position    { return p.Pos }
func (p *PHPDocNode) SetPos(pos Position) { p.Pos = pos }
func (p *PHPDocNode) String() string {
	return fmt.Sprintf("PHPDoc @ %d:%d", p.Pos.Line, p.Pos.Column)
}
func (p *PHPDocNode) TokenLiteral() string {
	return "/** ... */"
}

// PHPDocParam represents a parameter documented in PHPDoc
type PHPDocParam struct {
	Name        string
	Type        string
	Description string
}

// ParsePHPDoc parses a raw PHPDoc comment string and extracts structured information
func ParsePHPDoc(rawContent string) *PHPDocNode {
	phpdoc := &PHPDocNode{
		RawContent: rawContent,
		Params:     []PHPDocParam{},
	}

	// Remove the /** */ wrapper
	content := strings.TrimSpace(rawContent)
	if strings.HasPrefix(content, "/**") && strings.HasSuffix(content, "*/") {
		content = content[3 : len(content)-2]
	}

	lines := strings.Split(content, "\n")
	var descriptionLines []string
	var inDescription = true

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "*") {
			line = strings.TrimSpace(line[1:])
		}

		// Check for @param tags
		if strings.HasPrefix(line, "@param") {
			inDescription = false
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				param := PHPDocParam{
					Type: parts[1],
					Name: strings.TrimPrefix(parts[2], "$"),
				}
				if len(parts) > 3 {
					param.Description = strings.Join(parts[3:], " ")
				}
				phpdoc.Params = append(phpdoc.Params, param)
			}
		} else if strings.HasPrefix(line, "@return") {
			inDescription = false
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				phpdoc.ReturnType = parts[1]
				if len(parts) > 2 {
					// Could include description, but for now just capture the type
				}
			}
		} else if strings.HasPrefix(line, "@var") {
			inDescription = false
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				phpdoc.VarType = parts[1]
			}
		} else if strings.HasPrefix(line, "@") {
			// Any other @tag should stop description parsing
			inDescription = false
		} else if line != "" && inDescription {
			descriptionLines = append(descriptionLines, line)
		}
	}

	phpdoc.Description = strings.Join(descriptionLines, " ")
	return phpdoc
}

// ExtractPHPDocFromComment checks if a comment is a PHPDoc comment and parses it
func ExtractPHPDocFromComment(comment string) *PHPDocNode {
	comment = strings.TrimSpace(comment)
	if strings.HasPrefix(comment, "/**") && strings.HasSuffix(comment, "*/") {
		return ParsePHPDoc(comment)
	}
	return nil
}

// GetParamTypeFromPHPDoc finds the type for a parameter from PHPDoc
func (p *PHPDocNode) GetParamTypeFromPHPDoc(paramName string) string {
	for _, param := range p.Params {
		if param.Name == paramName {
			return param.Type
		}
	}
	return ""
}
