<?php

/** @param int<1, 10> $value */
function oneToTen(int $value): void {}

function wrongBranches(int $value): void
{
    if ($value > 10) { oneToTen($value); }
    if ($value < 1) { oneToTen($value); }
}
