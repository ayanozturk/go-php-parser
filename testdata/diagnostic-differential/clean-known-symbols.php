<?php

namespace App\Library;

function greet(string $name): string
{
    return $name;
}

namespace App;

use App\Library;

echo Library\greet('Codex');
