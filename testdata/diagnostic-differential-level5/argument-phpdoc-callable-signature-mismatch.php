<?php
/** @param callable(int): string $callback */
function invokeCallable(callable $callback): void
{
    acceptString($callback('wrong'));
}

function acceptString(string $value): void {}
