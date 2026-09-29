<?php

/** @param positive-int $value */
function positive(int $value): void {}
/** @param non-negative-int $value */
function nonNegative(int $value): void {}
/** @param negative-int $value */
function negative(int $value): void {}
/** @param non-positive-int $value */
function nonPositive(int $value): void {}

function greaterThanZero(int $value): void
{
    if ($value > 0) { positive($value); } else { nonPositive($value); }
}
function atLeastZero(int $value): void
{
    if ($value >= 0) { nonNegative($value); } else { negative($value); }
}
function lessThanZero(int $value): void
{
    if ($value < 0) { negative($value); } else { nonNegative($value); }
}
function atMostZero(int $value): void
{
    if ($value <= 0) { nonPositive($value); } else { positive($value); }
}
