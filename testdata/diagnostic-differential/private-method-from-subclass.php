<?php

class Base {
    private function hidden(): void {}
}

class Child extends Base {
    public function run(): void {
        $this->hidden();
    }
}
