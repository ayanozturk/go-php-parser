<?php
interface HasMethod
{
    public function available(): void;
}

interface FirstTag {}
interface SecondTag {}

function run((HasMethod&FirstTag)|(HasMethod&SecondTag) $value): void
{
    $value->available();
}
