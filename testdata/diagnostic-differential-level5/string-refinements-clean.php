<?php

/** @param non-empty-lowercase-string $value */
function acceptLowercase($value): void {}
/** @param numeric-string $value */
function acceptNumeric($value): void {}
/** @param non-falsy-string $value */
function acceptNonFalsy($value): void {}

function valid(): void
{
    $word = 'ready';
    acceptLowercase($word);
    acceptNumeric('12.5e2');
    acceptNonFalsy('0.0');
}
