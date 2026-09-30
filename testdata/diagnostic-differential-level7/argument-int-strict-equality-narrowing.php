<?php

/** @param int<5, 5> $value */
function acceptFive(int $value): void {}

function run(int $value): void
{
    acceptFive($value);

    if ($value === 5) {
        acceptFive($value);
    }

    if ($value !== 5) {
        return;
    }

    acceptFive($value);
}
