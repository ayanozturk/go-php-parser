<?php

/** @param positive-int $value */
function positive(int $value): void {}
/** @param non-positive-int $value */
function nonPositive(int $value): void {}

function wrongBranches(int $value): void
{
    if ($value > 0) { nonPositive($value); }
    if ($value <= 0) { positive($value); }
}
