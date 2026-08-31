<?php
interface HasMethod
{
    public function available(): void;
}

interface FirstTag {}
class MissingAlternative {}

function run((HasMethod&FirstTag)|MissingAlternative $value): void
{
    $value->available();
}
