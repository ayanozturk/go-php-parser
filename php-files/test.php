<?php

namespace App\MySpace;

class MyClass
{
    private string $name;

    public function __construct()
    {
        $this->name = 'MyName';
    }

    public function getName(): string
    {
        return $this->name;
    }

    public function setName(string $name): void
    {
        $this->name = $name;
    }
}