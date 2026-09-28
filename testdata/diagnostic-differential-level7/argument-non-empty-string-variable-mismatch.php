<?php

/** @param non-empty-string $value */
function acceptNonEmptyString($value): void {}

/** @param string $value */
function passUnrefinedString(string $value): void
{
    acceptNonEmptyString($value);
}
