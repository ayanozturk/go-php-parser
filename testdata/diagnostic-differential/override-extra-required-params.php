<?php
interface Contract {
    public function shape(string $name, int $count = 0): int;
}
class Bad implements Contract {
    public function shape(string $name, int $count, string $extra): int { return 0; }
}
