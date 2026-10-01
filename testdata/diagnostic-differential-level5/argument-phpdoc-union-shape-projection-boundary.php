<?php
/**
 * @param int|string $value
 * @return value-of<array{count: int}|array{label: string}>
 */
function projectedUnionValue(int|string $value): mixed { return $value; }

function checkUnionProjection(): void
{
    acceptScalar(projectedUnionValue(1));
}

function acceptScalar(int|string $value): void {}
