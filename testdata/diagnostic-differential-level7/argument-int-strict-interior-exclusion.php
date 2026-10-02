<?php

/** @param int<5, 6>|int<8, 10> $value */
function notSeven(int $value): void {}
/** @param int<7, 7> $value */
function exactlySeven(int $value): void {}

function unrefined(int $value): void
{
    notSeven($value);
}

function excluded(int $value): void
{
    if ($value >= 5 && $value <= 10) {
        if ($value !== 7) {
            notSeven($value);
        } else {
            exactlySeven($value);
        }
    }
}

function equalityElse(int $value): void
{
    if ($value >= 5 && $value <= 10) {
        if ($value === 7) {
            exactlySeven($value);
        } else {
            notSeven($value);
        }
    }
}
