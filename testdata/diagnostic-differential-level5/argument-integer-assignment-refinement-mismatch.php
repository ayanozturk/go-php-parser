<?php

/** @param int<5, 5> $value */
function exactlyFive(int $value): void {}

function mismatch(): void
{
    $literal = 4;
    exactlyFive($literal);
}
