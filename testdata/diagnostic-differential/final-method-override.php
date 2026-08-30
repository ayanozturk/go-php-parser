<?php

class Base {
    final public function sealed(): void {}
}

class Child extends Base {
    public function sealed(): void {}
}
