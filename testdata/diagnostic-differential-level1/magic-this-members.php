<?php
class MagicMembers {
    public function __call(string $name, array $arguments): mixed { return null; }
    public function __get(string $name): mixed { return null; }

    public function run(): void {
        $this->missingMethod();
        $value = $this->missingProperty;
    }
}
