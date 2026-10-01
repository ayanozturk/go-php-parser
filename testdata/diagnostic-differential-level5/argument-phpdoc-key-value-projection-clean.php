<?php
/**
 * @param int|string $value
 * @return value-of<array{foo: int, label: string}>
 */
function shapeValue(int|string $value): mixed { return $value; }

/** @return key-of<array{foo: int, label: string}> */
function shapeKey(): mixed { return 'foo'; }

/** @return value-of<array<int, string>> */
function genericArrayValue(): mixed { return 'value'; }

/** @return key-of<array<int, string>> */
function genericArrayKey(): mixed { return 0; }

function checkProjections(): void
{
    acceptScalar(shapeValue(1));
    acceptString(shapeKey());
    acceptString(genericArrayValue());
    acceptInt(genericArrayKey());
}

function acceptScalar(int|string $value): void {}
function acceptString(string $value): void {}
function acceptInt(int $value): void {}
