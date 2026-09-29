<?php

/** @param int<5, 5> $value */
function exactlyFive(int $value): void {}
/** @param int<1, 10> $value */
function oneToTen(int $value): void {}

function clean(): void
{
    $literal = 5;
    exactlyFive($literal);
    $literalCopy = $literal;
    exactlyFive($literalCopy);
}

function narrowed(int $value): void
{
    if ($value >= 1 && $value <= 10) {
        $copy = $value;
        oneToTen($copy);
    }
}
