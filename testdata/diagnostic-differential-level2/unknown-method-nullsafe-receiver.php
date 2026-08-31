<?php
class NullsafeService {}

function run(?NullsafeService $value): void
{
    $value?->missing();
}
