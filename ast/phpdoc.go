package ast

import (
	"fmt"
	"strings"
)

// PHPDocNode represents a parsed PHPDoc block
type PHPDocNode struct {
	RawContent         string
	Params             []PHPDocParam
	ReturnType         string
	VarType            string
	VarName            string
	Templates          []PHPDocTemplate
	TypeAliases        []PHPDocTypeAlias
	Extends            []PHPDocTypeReference
	Implements         []PHPDocTypeReference
	Deprecated         bool
	DeprecationMessage string
	Description        string
	Pos                Position
	EndPos             Position
}

// PHPDocTemplate describes a class or method template declaration such as
// @template T of EntityInterface.
type PHPDocTemplate struct {
	Name  string
	Bound string
}

// PHPDocTypeAlias describes a local @phpstan-type or @psalm-type binding.
type PHPDocTypeAlias struct {
	Name string
	Type string
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
		if tag, value, ok := phpDocTag(line); ok && isPHPDocParamTag(tag) {
			inDescription = false
			typeName, remainder := splitPHPDocParamTypeAndRest(value)
			parts := strings.Fields(remainder)
			if typeName != "" && len(parts) >= 1 {
				prefer := IsTemplateBindingParamType(typeName)
				if !prefer && tag != "param" {
					continue
				}
				upsertPHPDocParam(phpdoc, PHPDocParam{
					Type:        typeName,
					Name:        strings.TrimPrefix(parts[0], "$"),
					Description: strings.Join(parts[1:], " "),
				}, prefer)
			}
		} else if tag, value, ok := phpDocTag(line); ok && isPHPDocReturnTag(tag) {
			inDescription = false
			returnType, _ := splitPHPDocTypeAndRest(value)
			if phpdoc.ReturnType == "" || isTemplateUnionReturnType(returnType) {
				phpdoc.ReturnType = returnType
			}
		} else if strings.HasPrefix(line, "@var") {
			inDescription = false
			var remainder string
			phpdoc.VarType, remainder = splitPHPDocTypeAndRest(strings.TrimSpace(strings.TrimPrefix(line, "@var")))
			if fields := strings.Fields(remainder); len(fields) > 0 && strings.HasPrefix(fields[0], "$") {
				phpdoc.VarName = strings.TrimPrefix(fields[0], "$")
			}
		} else if tag, value, ok := phpDocTag(line); ok && isTemplateTag(tag) {
			inDescription = false
			if template, ok := parsePHPDocTemplate(value); ok {
				phpdoc.Templates = append(phpdoc.Templates, template)
			}
		} else if tag, value, ok := phpDocTag(line); ok && isTypeAliasTag(tag) {
			inDescription = false
			if alias, ok := parsePHPDocTypeAlias(value); ok {
				phpdoc.TypeAliases = append(phpdoc.TypeAliases, alias)
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
		} else if strings.HasPrefix(line, "@deprecated") {
			inDescription = false
			phpdoc.Deprecated = true
			phpdoc.DeprecationMessage = strings.TrimSpace(strings.TrimPrefix(line, "@deprecated"))
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

func splitPHPDocParamTypeAndRest(value string) (string, string) {
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
		case '$':
			if depth == 0 && (idx == 0 || value[idx-1] == ' ' || value[idx-1] == '\t') {
				return strings.TrimSpace(value[:idx]), strings.TrimSpace(value[idx:])
			}
		}
	}
	return splitPHPDocTypeAndRest(value)
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
				// Callable signatures conventionally permit whitespace after
				// their return separator: callable(): Result.
				if strings.HasSuffix(strings.TrimSpace(value[:idx]), ":") {
					continue
				}
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

func isPHPDocParamTag(tag string) bool {
	switch tag {
	case "param", "phpstan-param", "psalm-param":
		return true
	default:
		return false
	}
}

func isPHPDocReturnTag(tag string) bool {
	switch tag {
	case "return", "phpstan-return", "psalm-return":
		return true
	default:
		return false
	}
}

func upsertPHPDocParam(phpdoc *PHPDocNode, param PHPDocParam, preferTemplates bool) {
	if phpdoc == nil {
		return
	}
	for i, existing := range phpdoc.Params {
		if existing.Name != param.Name {
			continue
		}
		if preferTemplates {
			phpdoc.Params[i] = param
		}
		return
	}
	phpdoc.Params = append(phpdoc.Params, param)
}

func IsTemplateBindingParamType(typ string) bool {
	typ = strings.TrimSpace(strings.TrimPrefix(typ, "?"))
	if typ == "" || strings.ContainsAny(typ, `|&()`) {
		return false
	}
	lower := strings.ToLower(typ)
	if strings.HasPrefix(lower, "class-string<") {
		return true
	}
	return isBarePHPDocTemplateName(typ)
}

func isTemplateUnionReturnType(typ string) bool {
	typ = strings.TrimSpace(strings.TrimPrefix(typ, "?"))
	if typ == "" {
		return false
	}
	sawTemplate := false
	part := strings.Builder{}
	depth := 0
	flush := func() bool {
		token := strings.TrimSpace(part.String())
		part.Reset()
		if token == "" {
			return true
		}
		if strings.EqualFold(token, "null") {
			return true
		}
		if !isBarePHPDocTemplateName(token) {
			return false
		}
		sawTemplate = true
		return true
	}
	for _, r := range typ {
		switch r {
		case '<', '(', '{', '[':
			depth++
			part.WriteRune(r)
		case '>', ')', '}', ']':
			if depth > 0 {
				depth--
			}
			part.WriteRune(r)
		case '|':
			if depth == 0 {
				if !flush() {
					return false
				}
				continue
			}
			part.WriteRune(r)
		default:
			part.WriteRune(r)
		}
	}
	if !flush() {
		return false
	}
	return sawTemplate
}

func isBarePHPDocTemplateName(name string) bool {
	if name == "T" || name == "t" {
		return true
	}
	if len(name) < 2 || (name[0] != 'T' && name[0] != 't') {
		return false
	}
	if name[1] < 'A' || name[1] > 'Z' {
		return false
	}
	for i := 2; i < len(name); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' {
			continue
		}
		return false
	}
	return true
}

func isTemplateTag(tag string) bool {
	switch tag {
	case "template", "template-covariant", "template-contravariant", "phpstan-template", "psalm-template":
		return true
	default:
		return false
	}
}

func isTypeAliasTag(tag string) bool {
	switch tag {
	case "phpstan-type", "psalm-type":
		return true
	default:
		return false
	}
}

func parsePHPDocTypeAlias(value string) (PHPDocTypeAlias, bool) {
	value = strings.TrimSpace(value)
	name, rest := splitPHPDocTypeAndRest(value)
	if name == "" || rest == "" {
		return PHPDocTypeAlias{}, false
	}
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "=") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
	}
	aliasType, _ := splitPHPDocTypeAndRest(rest)
	if aliasType == "" {
		return PHPDocTypeAlias{}, false
	}
	return PHPDocTypeAlias{Name: name, Type: aliasType}, true
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
