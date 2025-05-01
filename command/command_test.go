package command

import (
	"bytes"
	"go-phpcs/ast"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCommandsMapIntegrity(t *testing.T) {
	expected := map[string]struct {
		Description string
	}{
		"ast":     {"Print the Abstract Syntax Tree"},
		"tokens":  {"Print the tokens from the lexer"},
		"style":   {"Check code style (e.g., function naming)"},
		"analyse": {"Static analysis: unknown function calls (PoC)"},
	}
	for name, meta := range expected {
		cmd, ok := Commands[name]
		if !ok {
			t.Errorf("Command %q not found in Commands map", name)
			continue
		}
		if cmd.Name != name {
			t.Errorf("Command %q has Name %q (want %q)", name, cmd.Name, name)
		}
		if !strings.Contains(cmd.Description, meta.Description) {
			t.Errorf("Command %q has Description %q (want it to contain %q)", name, cmd.Description, meta.Description)
		}
		if cmd.Execute == nil {
			t.Errorf("Command %q has nil Execute function", name)
		}
	}
}

func TestPrintUsage_Output(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintUsage()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	for name := range Commands {
		if !strings.Contains(output, name) {
			t.Errorf("PrintUsage output missing command %q", name)
		}
	}
	if !strings.Contains(output, "Usage:") {
		t.Error("PrintUsage output missing usage line")
	}
}

func TestCommandStructFields(t *testing.T) {
	cmd := Command{
		Name:        "test",
		Description: "desc",
		Execute: func(nodes []ast.Node, filename string, w io.Writer) {},
	}
	if cmd.Name != "test" {
		t.Errorf("unexpected Name: %q", cmd.Name)
	}
	if cmd.Description != "desc" {
		t.Errorf("unexpected Description: %q", cmd.Description)
	}
	if cmd.Execute == nil {
		t.Error("Execute should not be nil")
	}
}
