package command_test

import (
	"testing"

	"github.com/ayanozturk/go-php-parser/analyse"
	"github.com/ayanozturk/go-php-parser/ast"
	"github.com/ayanozturk/go-php-parser/command"
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
	nodesS := command.ParseNodesForIndex([]byte(src))

	idxS := analyse.BuildProjectIndex(map[string][]ast.Node{"f.php": nodesS})

	classS, okS := idxS.ResolveClass(`App\User`)
	if !okS {
		t.Fatalf("ResolveClass App\\User: ok=%v", okS)
	}
	if classS.Name != `App\User` {
		t.Fatalf("class name=%q want App\\User", classS.Name)
	}

	mS, okS := idxS.ResolveMethod(`App\User`, "id")
	if !okS {
		t.Fatalf("ResolveMethod id: ok=%v", okS)
	}
	if mS.ReturnType != "int" {
		t.Fatalf("id return=%q want int", mS.ReturnType)
	}
}
