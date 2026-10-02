<?php

/** @param 'red' $color */
function acceptRed($color): void {}
/** @param int<1, 10> $value */
function acceptBounded(int $value): void {}

/** @param 'red'|'blue' $color */
function invalidDisjunction($color): void
{
    if ($color === 'red' || $color === 'blue') {
        acceptRed($color);
    }
}

function invalidIntegerJoin(bool $chooseLow): void
{
    if ($chooseLow) { $value = 1; } else { $value = 11; }
    acceptBounded($value);
}
