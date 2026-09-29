<?php

/** @param positive-int $value */
function acceptPositive(int $value): void {}
/** @param non-negative-int $value */
function acceptNonNegative(int $value): void {}

function run(): void
{
    acceptPositive(42);
    acceptNonNegative(0);
}
