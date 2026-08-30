<?php

class ParentClass {
    public function greet(string $name): string {
        return $name;
    }
}

class ChildClass extends ParentClass {
    public function run(): string {
        return $this->greet('Codex');
    }
}
