<?php

namespace PHPUnit\Framework\MockObject;

interface MockObject
{
    public function method(string $name): MockObject;
}

namespace {

use PHPUnit\Framework\MockObject\MockObject;

class User
{
    public function id(): string
    {
        return '';
    }
}

function run(MockObject&User $user): void
{
    $user->method('id');
    $user->id();
}

}
