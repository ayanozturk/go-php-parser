<?php
/**
 * @phpstan-consistent-constructor
 */
class Base {
    public function __construct(string $name) {}
}

class Child extends Base {
    protected function __construct(string $name, int $id) {}
}
