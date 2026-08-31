<?php
class NullableService {}

function run(?NullableService $value): void
{
    $value->missing();
}
