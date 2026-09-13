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
