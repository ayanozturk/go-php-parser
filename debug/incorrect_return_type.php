<?php

class Example {
    // Declared return type is int, but returns a string
    public function getValue(): int {
        return "not an int";
    }

    // Declared return type is string, but returns an int
    public function getString(): string {
        return 123;
    }

    // No return type declared, but returns a value
    public function getSomething() {
        return true;
    }
}
