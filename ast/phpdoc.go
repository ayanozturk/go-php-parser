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

	lines := logicalPHPDocLines(content)
	var descriptionLines []string
	var inDescription = true

	for _, line := range lines {
		// Parse the leading @tag at most once per line. The previous
		// if/else-if chain re-ran phpDocTag (and strings.Fields) on every
		// failed branch, so description / unknown-tag lines paid Fields up
		// to six times — the dominant LowerFile PHPDoc CPU under WP.
		if tag, value, ok := phpDocTag(line); ok {
			inDescription = false
			switch {
			case isPHPDocParamTag(tag):
				typeName, remainder := splitPHPDocParamTypeAndRest(value)
				name, desc, hasName := firstFieldRest(remainder)
				if typeName != "" && hasName {
					prefer := IsTemplateBindingParamType(typeName)
					if !prefer && tag != "param" {
						break
					}
					upsertPHPDocParam(phpdoc, PHPDocParam{
						Type:        typeName,
						Name:        strings.TrimPrefix(name, "$"),
						Description: desc,
					}, prefer)
				}
			case isPHPDocReturnTag(tag):
				returnType, _ := splitPHPDocTypeAndRest(value)
				if phpdoc.ReturnType == "" || isTemplateUnionReturnType(returnType) {
					phpdoc.ReturnType = returnType
				}
			case tag == "var":
				var remainder string
				phpdoc.VarType, remainder = splitPHPDocTypeAndRest(value)
				if name, _, ok := firstFieldRest(remainder); ok && strings.HasPrefix(name, "$") {
					phpdoc.VarName = strings.TrimPrefix(name, "$")
				}
			case isTemplateTag(tag):
				if template, ok := parsePHPDocTemplate(value); ok {
					phpdoc.Templates = append(phpdoc.Templates, template)
				}
			case isTypeAliasTag(tag):
				if alias, ok := parsePHPDocTypeAlias(value); ok {
					phpdoc.TypeAliases = append(phpdoc.TypeAliases, alias)
				}
			case isExtendsTag(tag):
				if ref, ok := parsePHPDocTypeReference(value); ok {
					phpdoc.Extends = append(phpdoc.Extends, ref)
				}
			case isImplementsTag(tag):
				if ref, ok := parsePHPDocTypeReference(value); ok {
					phpdoc.Implements = append(phpdoc.Implements, ref)
				}
			case tag == "deprecated":
				phpdoc.Deprecated = true
				phpdoc.DeprecationMessage = value
			default:
				// Any other recognized @tag stops description parsing.
			}
			continue
		}
		if strings.HasPrefix(line, "@deprecated") {
			// Bare @deprecated (no value) fails phpDocTag's "tag + value"
			// rule; keep the classic HasPrefix path for that case.
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

func logicalPHPDocLines(content string) []string {
	physical := strings.Split(content, "\n")
	lines := make([]string, 0, len(physical))
	for index := 0; index < len(physical); index++ {
		line := strings.TrimSpace(physical[index])
		if strings.HasPrefix(line, "*") {
			line = strings.TrimSpace(line[1:])
		}
		combined := line
		joinType := phpDocLineStartsTypeTag(line)
		for joinType && phpDocTypeDelimiterDepth(combined) > 0 && index+1 < len(physical) {
			index++
			continuation := strings.TrimSpace(physical[index])
			if strings.HasPrefix(continuation, "*") {
				continuation = strings.TrimSpace(continuation[1:])
			}
			combined = strings.TrimSpace(combined + " " + continuation)
		}
		lines = append(lines, splitCompoundPHPDocTagLines(combined)...)
	}
	return lines
}

func splitCompoundPHPDocTagLines(line string) []string {
	if !strings.Contains(line, "@") {
		return []string{line}
	}
	var parts []string
	depth := 0
	start := -1
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '<', '(', '{', '[':
			depth++
		case '>', ')', '}', ']':
			if depth > 0 {
				depth--
			}
		}
		if depth != 0 || line[i] != '@' {
			continue
		}
		if i > 0 && line[i-1] != ' ' && line[i-1] != '\t' {
			continue
		}
		if start >= 0 {
			parts = append(parts, strings.TrimSpace(line[start:i]))
		}
		start = i
	}
	if start < 0 {
		return []string{line}
	}
	if tail := strings.TrimSpace(line[start:]); tail != "" {
		parts = append(parts, tail)
	}
	if len(parts) <= 1 {
		return []string{line}
	}
	return parts
}

func phpDocLineStartsTypeTag(line string) bool {
	tag, _, ok := phpDocTag(line)
	if !ok {
		return false
	}
	return isPHPDocParamTag(tag) || isPHPDocReturnTag(tag) || tag == "var" || isTypeAliasTag(tag) || isExtendsTag(tag) || isImplementsTag(tag)
}

func phpDocTypeDelimiterDepth(value string) int {
	depth := 0
	for _, r := range value {
		switch r {
		case '<', '(', '{', '[':
			depth++
		case '>', ')', '}', ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return depth
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
	// Match strings.Fields semantics without allocating on the common
	// non-tag path (description prose): reject before any Fields/Join.
	i := 0
	n := len(line)
	for i < n && isPHPDocASCIISpace(line[i]) {
		i++
	}
	if i >= n || line[i] != '@' {
		return "", "", false
	}
	tagStart := i + 1
	i = tagStart
	for i < n && !isPHPDocASCIISpace(line[i]) {
		i++
	}
	if i == tagStart {
		return "", "", false
	}
	tagEnd := i
	for i < n && isPHPDocASCIISpace(line[i]) {
		i++
	}
	if i >= n {
		// No value field — same as len(Fields) < 2.
		return "", "", false
	}
	tag := strings.ToLower(line[tagStart:tagEnd])
	value := joinFieldsLike(line[i:])
	return tag, value, true
}

func isPHPDocASCIISpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	default:
		return false
	}
}

// joinFieldsLike matches strings.Join(strings.Fields(s), " ") for ASCII
// whitespace. Fast path returns s unchanged when already Fields-normalized.
func joinFieldsLike(s string) string {
	start := 0
	n := len(s)
	for start < n && isPHPDocASCIISpace(s[start]) {
		start++
	}
	if start >= n {
		return ""
	}
	end := n
	for end > start && isPHPDocASCIISpace(s[end-1]) {
		end--
	}
	needCollapse := start > 0 || end < n
	if !needCollapse {
		for i := start; i < end; i++ {
			if !isPHPDocASCIISpace(s[i]) {
				continue
			}
			if s[i] != ' ' || i+1 < end && isPHPDocASCIISpace(s[i+1]) {
				needCollapse = true
				break
			}
		}
	}
	if !needCollapse {
		return s[start:end]
	}
	var b strings.Builder
	b.Grow(end - start)
	inSpace := false
	wrote := false
	for i := start; i < end; i++ {
		if isPHPDocASCIISpace(s[i]) {
			inSpace = true
			continue
		}
		if inSpace && wrote {
			b.WriteByte(' ')
		}
		inSpace = false
		wrote = true
		b.WriteByte(s[i])
	}
	return b.String()
}

// firstFieldRest returns the first Fields-like token and the remainder joined
// with single spaces (same as Fields[0] / Join(Fields[1:], " ")).
func firstFieldRest(s string) (field, rest string, ok bool) {
	start := 0
	n := len(s)
	for start < n && isPHPDocASCIISpace(s[start]) {
		start++
	}
	if start >= n {
		return "", "", false
	}
	end := start
	for end < n && !isPHPDocASCIISpace(s[end]) {
		end++
	}
	field = s[start:end]
	if end >= n {
		return field, "", true
	}
	return field, joinFieldsLike(s[end:]), true
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
