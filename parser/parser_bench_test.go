package parser

import (
	"go-phpcs/lexer"
	"testing"
)

const benchPHPCode = `<?php
namespace App\Controllers;

use App\Models\User;
use App\Models\Post;

class UserController extends BaseController {
    private ?User $user;
    protected string $title;

    public function __construct(User $user, string $title = 'Default') {
        $this->user = $user;
        $this->title = $title;
    }

    public function show(int $id): string {
        $post = Post::find($id);
        if ($post === null) {
            return "Post not found";
        }

        $result = "";
        for ($i = 0; $i < 10; $i++) {
            $result .= "Line " . $i;
        }

        switch ($id) {
            case 1:
                return "Admin Post";
            default:
                return "User Post";
        }
    }
}
`

func BenchmarkParse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(benchPHPCode)
		p := New(l, false)
		_ = p.Parse()
	}
}

func BenchmarkParseSkipFunctionBodies(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := lexer.New(benchPHPCode)
		p := New(l, false)
		p.SkipFunctionBodies = true
		_ = p.Parse()
	}
}
