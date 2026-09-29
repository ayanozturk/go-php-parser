<?php

/** @param int<1, 10> $value */
function acceptOneToTen(int $value): void {}

/** @param int $value */
function passUnknownInt(int $value): void
{
    acceptOneToTen($value);
}
