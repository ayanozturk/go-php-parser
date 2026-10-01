<?php
/** @param callable(int): string $callback */
function invokeCallable(callable $callback): void
{
    acceptString($callback(1));
}

/** @param Closure(int): string $callback */
function invokeClosure(Closure $callback): void
{
    acceptString($callback(1));
}

function acceptString(string $value): void {}
