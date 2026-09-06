<?php

function take(string $n): void
{
}

function run(mixed $x): void
{
    take(is_string($x) && $x !== '' ? (int)$x : null);
}
