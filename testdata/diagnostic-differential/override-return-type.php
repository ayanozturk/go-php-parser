<?php
interface Contract {
    public function shape(string $name): int;
}
class Bad implements Contract {
    public function shape(string $name): string { return "x"; }
}

