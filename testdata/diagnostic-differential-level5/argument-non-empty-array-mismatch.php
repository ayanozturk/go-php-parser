<?php

/** @param non-empty-array<int, int> $values */
function acceptNonEmptyArray(array $values): void {}

function run(): void
{
    acceptNonEmptyArray([]);
}
