<?php

class Calls {
    public function takesOne($value): void {}

    public function run(): void {
        $this->takesOne();
    }
}
