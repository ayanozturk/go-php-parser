<?php
class ConstructorUsage {
    public function __construct(int $unused, int $used) { $this->value = $used; }
    private int $value;
}

class PromotedConstructorParameter {
    public function __construct(private int $value) {}
}

class UsedConstructorParameter {
    private int $value;
    public function __construct(int $value) { $this->value = $value; }
}
