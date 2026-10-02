<?php
/**
 * @param positive-int $number
 */
function checkKnown(string $value, $number): void {
    isset($value);
    $value ?? 'fallback';
    empty($number);
}

function checkNullable(?string $value): void {
    isset($value);
    $value ?? 'fallback';
    empty($value);
}
