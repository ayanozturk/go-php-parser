<?php
/**
 * @template T of int|float
 * @param T $value
 */
function acceptNumber(int|float $value): void {}
/**
 * @template T
 * @param T $value
 */
function acceptAny($value): void {}
acceptNumber(1);
acceptNumber(1.5);
acceptNumber('wrong');
acceptAny('any type');
