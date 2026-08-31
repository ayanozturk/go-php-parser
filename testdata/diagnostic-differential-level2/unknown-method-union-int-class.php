<?php
class UnionIntService {}

function run(int|UnionIntService $value): void
{
    $value->missing();
}
