<?php

/** @param positive-int $value */
function acceptPositive(int $value): void {}

/** @param int $value */
function passUnknownInt(int $value): void
{
    acceptPositive($value);
}
