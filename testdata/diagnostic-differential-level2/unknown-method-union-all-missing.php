<?php
class FirstMissing {}
class SecondMissing {}

function run(FirstMissing|SecondMissing $value): void
{
    $value->missing();
}
