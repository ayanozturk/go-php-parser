package ast

import (
	"strings"

	"github.com/ayanozturk/go-php-parser/token"
)

// Modifier mirrors one entry in syntax.KindModifierList: a keyword token, or an
// asymmetric visibility group like public(set).
type Modifier struct {
	Tok  token.TokenType
	Set  bool   // true => Tok(set), e.g. private(set)
	Text string // original spelling; empty uses the canonical lower keyword
}

// ModifierList is the structured replacement for string Visibility / Modifiers /
// SetVisibility on declarations.
type ModifierList []Modifier

// ModifierFromText parses a legacy modifier spelling into a Modifier.
// Accepts "public", "PRIVATE", "protected(set)", etc.
func ModifierFromText(s string) Modifier {
	s = strings.TrimSpace(s)
	set := false
	if strings.HasSuffix(strings.ToLower(s), "(set)") {
		set = true
		s = s[:len(s)-5]
	}
	lower := strings.ToLower(s)
	m := Modifier{Text: s, Set: set}
	switch lower {
	case "public":
		m.Tok = token.T_PUBLIC
	case "protected":
		m.Tok = token.T_PROTECTED
	case "private":
		m.Tok = token.T_PRIVATE
	case "static":
		m.Tok = token.T_STATIC
	case "final":
		m.Tok = token.T_FINAL
	case "abstract":
		m.Tok = token.T_ABSTRACT
	case "readonly":
		m.Tok = token.T_READONLY
	}
	return m
}

// ModifierListFromTexts builds a ModifierList from legacy string spellings.
func ModifierListFromTexts(texts []string) ModifierList {
	if len(texts) == 0 {
		return nil
	}
	out := make(ModifierList, 0, len(texts))
	for _, t := range texts {
		if strings.TrimSpace(t) == "" {
			continue
		}
		out = append(out, ModifierFromText(t))
	}
	return out
}

func (m Modifier) canonicalBase() string {
	if m.Text != "" {
		return strings.ToLower(m.Text)
	}
	switch m.Tok {
	case token.T_PUBLIC:
		return "public"
	case token.T_PROTECTED:
		return "protected"
	case token.T_PRIVATE:
		return "private"
	case token.T_STATIC:
		return "static"
	case token.T_FINAL:
		return "final"
	case token.T_ABSTRACT:
		return "abstract"
	case token.T_READONLY:
		return "readonly"
	default:
		return ""
	}
}

// Canonical returns the lower keyword, with "(set)" when asymmetric.
func (m Modifier) Canonical() string {
	base := m.canonicalBase()
	if base == "" {
		return ""
	}
	if m.Set {
		return base + "(set)"
	}
	return base
}

// Has reports whether a non-set modifier of the given token kind is present.
func (ml ModifierList) Has(tt token.TokenType) bool {
	for _, m := range ml {
		if !m.Set && m.Tok == tt {
			return true
		}
	}
	return false
}

// HasName reports whether any modifier matches the spelling (case-insensitive),
// including asymmetric forms like "private(set)".
func (ml ModifierList) HasName(name string) bool {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return false
	}
	for _, m := range ml {
		if m.Canonical() == want {
			return true
		}
	}
	return false
}

// Visibility returns the get/read visibility keyword, or "" if unset.
func (ml ModifierList) Visibility() string {
	for _, m := range ml {
		if m.Set {
			continue
		}
		switch m.Tok {
		case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE:
			return m.canonicalBase()
		}
	}
	return ""
}

// SetVisibility returns the asymmetric write visibility, or "" if unset.
func (ml ModifierList) SetVisibility() string {
	for _, m := range ml {
		if !m.Set {
			continue
		}
		switch m.Tok {
		case token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE:
			return m.canonicalBase()
		}
	}
	return ""
}

// Texts returns canonical spellings for debug / gradual call-site migration.
func (ml ModifierList) Texts() []string {
	if len(ml) == 0 {
		return nil
	}
	out := make([]string, 0, len(ml))
	for _, m := range ml {
		if c := m.Canonical(); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// String joins canonical modifiers with spaces.
func (ml ModifierList) String() string {
	return strings.Join(ml.Texts(), " ")
}

// DefaultVisibility returns Visibility() or "public" when empty.
func (ml ModifierList) DefaultVisibility() string {
	if v := ml.Visibility(); v != "" {
		return v
	}
	return "public"
}
