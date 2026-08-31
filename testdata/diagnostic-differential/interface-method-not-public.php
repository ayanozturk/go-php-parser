<?php
interface Contract {
    public function mustBePublic(): void;
}
class Impl implements Contract {
    protected function mustBePublic(): void {}
}

