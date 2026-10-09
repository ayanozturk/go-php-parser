<?php
namespace Captures;
/**
 * @template Result
 * @param callable(): Result $factory
 * @return Result
 */
function evaluate(callable $factory) { return $factory(); }
function acceptInt(int $value): void {}
function acceptString(string $value): void {}
function run(int $number): int {
    $text = 'ready';
    $callback = function () use ($number) { return $number; };
    acceptInt(evaluate($callback));
    acceptString(evaluate(fn() => $text));
    $shadow = fn(int $text) => acceptInt($text);
    $modified = function () use ($text) { $text = 3; return $text; };
    acceptInt(evaluate($modified));
    acceptString($text);
    return evaluate(fn() => $number);
}
