<?php
class TakesOne {
    private int $value;
    public function __construct(int $value) { $this->value = $value; }
    public function run(int $value): void { $this->value = $value; }
}

$instance = new TakesOne(1, 2);
$instance->run(1, 2);
