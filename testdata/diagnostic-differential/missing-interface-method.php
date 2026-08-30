<?php
interface Contract {
    public function required(): void;
}
class MissingMethods implements Contract {}

