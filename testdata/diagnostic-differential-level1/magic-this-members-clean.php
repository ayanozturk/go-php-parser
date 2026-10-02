<?php
class DeclaredMembers {
    public function run(): int { return $this->value(); }
    private function value(): int { return 1; }
}
