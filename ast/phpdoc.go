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
	Templates   []PHPDocTemplate
	Extends     []PHPDocTypeReference
	Implements  []PHPDocTypeReference
	Description string
	Pos         Position
	EndPos      Position
}

// PHPDocTemplate describes a class or method template declaration such as
// @template T of EntityInterface.
type PHPDocTemplate struct {
	Name  string
	Bound string
}

// PHPDocTypeReference describes a generic inheritance annotation such as
// @extends Repository<User>.
type PHPDocTypeReference struct {
	Name          string
	TypeArguments []string
}

func (p *PHPDocNode) NodeType() string       { return "PHPDoc" }
func (p *PHPDocNode) GetPos() Position       { return p.Pos }
func (p *PHPDocNode) SetPos(pos Position)    { p.Pos = pos }
func (p *PHPDocNode) GetEndPos() Position    { return p.EndPos }
func (p *PHPDocNode) SetEndPos(pos Position) { p.EndPos = pos }
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
	if len(content) >= len("/**")+len("*/") && strings.HasPrefix(content, "/**") && strings.HasSuffix(content, "*/") {
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
			typeName, remainder := splitPHPDocTypeAndRest(strings.TrimSpace(strings.TrimPrefix(line, "@param")))
			parts := strings.Fields(remainder)
			if typeName != "" && len(parts) >= 1 {
				param := PHPDocParam{
					Type: typeName,
					Name: strings.TrimPrefix(parts[0], "$"),
				}
				if len(parts) > 1 {
					param.Description = strings.Join(parts[1:], " ")
				}
				phpdoc.Params = append(phpdoc.Params, param)
			}
		} else if strings.HasPrefix(line, "@return") {
			inDescription = false
			phpdoc.ReturnType, _ = splitPHPDocTypeAndRest(strings.TrimSpace(strings.TrimPrefix(line, "@return")))
		} else if strings.HasPrefix(line, "@var") {
			inDescription = false
			phpdoc.VarType, _ = splitPHPDocTypeAndRest(strings.TrimSpace(strings.TrimPrefix(line, "@var")))
		} else if tag, value, ok := phpDocTag(line); ok && isTemplateTag(tag) {
			inDescription = false
			if template, ok := parsePHPDocTemplate(value); ok {
				phpdoc.Templates = append(phpdoc.Templates, template)
			}
		} else if tag, value, ok := phpDocTag(line); ok && isExtendsTag(tag) {
			inDescription = false
			if ref, ok := parsePHPDocTypeReference(value); ok {
				phpdoc.Extends = append(phpdoc.Extends, ref)
			}
		} else if tag, value, ok := phpDocTag(line); ok && isImplementsTag(tag) {
			inDescription = false
			if ref, ok := parsePHPDocTypeReference(value); ok {
				phpdoc.Implements = append(phpdoc.Implements, ref)
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

func splitPHPDocTypeAndRest(value string) (string, string) {
	value = strings.TrimSpace(value)
	depth := 0
	for idx, r := range value {
		switch r {
		case '<', '(', '{', '[':
			depth++
		case '>', ')', '}', ']':
			if depth > 0 {
				depth--
			}
		case ' ', '\t':
			if depth == 0 {
				return strings.TrimSpace(value[:idx]), strings.TrimSpace(value[idx:])
			}
		}
	}
	return value, ""
}

func phpDocTag(line string) (string, string, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 || !strings.HasPrefix(parts[0], "@") {
		return "", "", false
	}
	return strings.ToLower(strings.TrimPrefix(parts[0], "@")), strings.Join(parts[1:], " "), true
}

func isTemplateTag(tag string) bool {
	switch tag {
	case "template", "template-covariant", "template-contravariant", "phpstan-template", "psalm-template":
		return true
	default:
		return false
	}
}

func isExtendsTag(tag string) bool {
	switch tag {
	case "extends", "template-extends", "phpstan-extends", "psalm-extends":
		return true
	default:
		return false
	}
}

func isImplementsTag(tag string) bool {
	switch tag {
	case "implements", "template-implements", "phpstan-implements", "psalm-implements":
		return true
	default:
		return false
	}
}

func parsePHPDocTemplate(value string) (PHPDocTemplate, bool) {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return PHPDocTemplate{}, false
	}
	template := PHPDocTemplate{Name: parts[0]}
	if len(parts) >= 3 && (strings.EqualFold(parts[1], "of") || strings.EqualFold(parts[1], "as")) {
		template.Bound = parts[2]
	}
	return template, template.Name != ""
}

func parsePHPDocTypeReference(value string) (PHPDocTypeReference, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return PHPDocTypeReference{}, false
	}
	// Ignore a trailing prose description while preserving whitespace inside
	// nested generic expressions.
	depth := 0
	for idx, r := range value {
		switch r {
		case '<', '(', '{', '[':
			depth++
		case '>', ')', '}', ']':
			if depth > 0 {
				depth--
			}
		case ' ', '\t':
			if depth == 0 {
				value = value[:idx]
				goto parsedType
			}
		}
	}
parsedType:
	open := strings.Index(value, "<")
	if open < 0 {
		return PHPDocTypeReference{Name: value}, true
	}
	if !strings.HasSuffix(value, ">") || open == 0 {
		return PHPDocTypeReference{}, false
	}
	ref := PHPDocTypeReference{Name: strings.TrimSpace(value[:open])}
	for _, argument := range splitPHPDocGenericArguments(value[open+1 : len(value)-1]) {
		if argument = strings.TrimSpace(argument); argument != "" {
			ref.TypeArguments = append(ref.TypeArguments, argument)
		}
	}
	return ref, ref.Name != ""
}

func splitPHPDocGenericArguments(raw string) []string {
	start, depth := 0, 0
	var parts []string
	for idx, r := range raw {
		switch r {
		case '<', '(', '{', '[':
			depth++
		case '>', ')', '}', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, raw[start:idx])
				start = idx + 1
			}
		}
	}
	return append(parts, raw[start:])
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
