<?php
/**
 * @param callable(int): string $callback
 * @param mixed $value
 */
function invokeWithUnknownValue(callable $callback, mixed $value): void
{
    acceptString($callback($value));
}

function acceptString(string $value): void {}
