<?php

/** @param int<1, 10> $value */
function oneToTen(int $value): void {}
/** @param int<-10, -1> $value */
function minusTenToMinusOne(int $value): void {}

function bounded(int $value): void
{
    if ($value >= 1 && $value <= 10) { oneToTen($value); }
    if ($value <= -1 && $value >= -10) { minusTenToMinusOne($value); }
    if (1 <= $value && 10 >= $value) { oneToTen($value); }
}

function falseBranch(int $value): void
{
    if ($value > 10) { return; } else { oneToTen($value); }
}
