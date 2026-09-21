package ast

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/token"
)

func TestModifierListVisibilityAndSet(t *testing.T) {
	ml := ModifierListFromTexts([]string{"public", "private(set)", "readonly"})
	if ml.Visibility() != "public" {
		t.Fatalf("Visibility=%q", ml.Visibility())
	}
	if ml.SetVisibility() != "private" {
		t.Fatalf("SetVisibility=%q", ml.SetVisibility())
	}
	if !ml.Has(token.T_READONLY) || !ml.HasName("private(set)") {
		t.Fatalf("Has checks failed: %v", ml.Texts())
	}
}

func TestModifierListEdgeCases(t *testing.T) {
	if got := ModifierList(nil).DefaultVisibility(); got != "public" {
		t.Fatalf("empty DefaultVisibility=%q", got)
	}
	if got := (ModifierList{}).DefaultVisibility(); got != "public" {
		t.Fatalf("empty slice DefaultVisibility=%q", got)
	}
	if got := ModifierListFromTexts([]string{"protected"}).DefaultVisibility(); got != "protected" {
		t.Fatalf("DefaultVisibility=%q", got)
	}

	if got := ModifierListFromTexts(nil); got != nil {
		t.Fatalf("FromTexts(nil)=%v", got)
	}
	if got := ModifierListFromTexts([]string{"", "  ", "static"}); len(got) != 1 || !got.Has(token.T_STATIC) {
		t.Fatalf("blank entries not skipped: %v", got)
	}

	for _, tt := range []token.TokenType{
		token.T_PUBLIC, token.T_PROTECTED, token.T_PRIVATE,
		token.T_STATIC, token.T_FINAL, token.T_ABSTRACT, token.T_READONLY,
	} {
		m := Modifier{Tok: tt}
		if m.Canonical() == "" {
			t.Fatalf("expected Canonical for tok %v", tt)
		}
	}
	setTokOnly := Modifier{Tok: token.T_PRIVATE, Set: true}
	if got := setTokOnly.Canonical(); got != "private(set)" {
		t.Fatalf("set Tok-only Canonical=%q", got)
	}
	if got := (Modifier{}).Canonical(); got != "" {
		t.Fatalf("empty Canonical=%q", got)
	}

	ml := ModifierListFromTexts([]string{"private(set)", "final"})
	if ml.Has(token.T_PRIVATE) {
		t.Fatalf("Has(T_PRIVATE) should ignore asymmetric set modifiers")
	}
	if ml.HasName("") {
		t.Fatalf("HasName(\"\") should be false")
	}
	if ml.Visibility() != "" {
		t.Fatalf("Visibility with only set-mod should be empty, got %q", ml.Visibility())
	}
	if ml.SetVisibility() != "private" {
		t.Fatalf("SetVisibility=%q", ml.SetVisibility())
	}

	unknown := ModifierFromText("weird")
	if unknown.Tok != 0 || unknown.Canonical() != "weird" {
		t.Fatalf("unknown modifier: %+v Canonical=%q", unknown, unknown.Canonical())
	}
	cased := ModifierFromText("PROTECTED(set)")
	if !cased.Set || cased.Tok != token.T_PROTECTED || cased.Canonical() != "protected(set)" {
		t.Fatalf("cased set modifier: %+v Canonical=%q", cased, cased.Canonical())
	}
}
