<?php

/** @param value-of<array{count: int, label: string}> $value */
function acceptProjectedValue($value): void {}

function runProjectedValues(): void
{
    acceptProjectedValue(true);
    acceptProjectedValue([]);
}
