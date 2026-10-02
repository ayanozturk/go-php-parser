<?php

/** @param 'red' $color */
function acceptRed($color): void {}
/** @param int<1, 10> $value */
function acceptBounded(int $value): void {}

/** @param 'red'|'blue' $color */
function boundaryDisjunction($color, bool $enabled): void
{
    if ($color === 'red' || $enabled) {
        acceptRed($color);
    }
}

function boundaryIntegerJoin(bool $enabled, int $input): void
{
    if ($enabled) { $value = 1; } else { $value = $input; }
    acceptBounded($value);
}
