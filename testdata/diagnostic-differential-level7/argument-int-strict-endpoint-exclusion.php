<?php

/** @param int<6, 10> $value */
function aboveFive(int $value): void {}
/** @param int<5, 9> $value */
function belowTen(int $value): void {}
/** @param int<5, 5> $value */
function exactlyFive(int $value): void {}
/** @param int<10, 10> $value */
function exactlyTen(int $value): void {}

function checkLower(int $value): void
{
    aboveFive($value);

    if ($value >= 5 && $value <= 10) {
        if ($value !== 5) {
            aboveFive($value);
        } else {
            exactlyFive($value);
        }
    }
}

function checkUpper(int $value): void
{
    if ($value >= 5 && $value <= 10) {
        if ($value === 10) {
            exactlyTen($value);
        } else {
            belowTen($value);
        }
    }
}
