<?php
class UnionStringService {}

function run(UnionStringService|string $value): void
{
    $value->missing();
}
