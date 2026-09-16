package command_test

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/command"
	"github.com/ayanozturk/go-php-parser/lexer"
	"github.com/ayanozturk/go-php-parser/parser"
)

func TestParseNodesForIndexProjectIndexParity(t *testing.T) {
	src := `<?php
namespace App;
use Vendor\Base;
class User extends Base {
    public string $name;
    public function id(): int {}
    private function set_name(string $n): void {}
}
`
	classic := parser.New(lexer.New(src), false)
	classic.SkipFunctionBodies = true
	nodesC := classic.Parse()
	nodesS := command.ParseNodesForIndex([]byte(src))

	idxC := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesC})
	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	_, okC := idxC.ResolveClass(`App\User`)
	_, okS := idxS.ResolveClass(`App\User`)
	if !okC || !okS {
		t.Fatalf("ResolveClass classic=%v syntax=%v", okC, okS)
	}
	mC, okC := idxC.ResolveMethod(`App\User`, "id")
	mS, okS := idxS.ResolveMethod(`App\User`, "id")
	if !okC || !okS || mC.ReturnType != mS.ReturnType {
		t.Fatalf("id return classic=%q/%v syntax=%q/%v", mC.ReturnType, okC, mS.ReturnType, okS)
	}
}
