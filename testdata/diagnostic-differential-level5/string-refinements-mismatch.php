<?php

/** @param non-empty-lowercase-string $value */
function acceptLowercase($value): void {}
/** @param numeric-string $value */
function acceptNumeric($value): void {}
/** @param non-falsy-string $value */
function acceptNonFalsy($value): void {}

function invalid(): void
{
    acceptLowercase('Ready');
    acceptLowercase('');
    acceptNumeric('12px');
    acceptNonFalsy('0');
}
