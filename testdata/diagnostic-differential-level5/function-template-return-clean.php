<?php
/**
 * @template Value
 * @param Value $value
 * @return Value
 */
function preserve($value) { return $value; }
/**
 * @template Result
 * @param callable(): Result $factory
 * @return Result
 */
function evaluate(callable $factory) { return $factory(); }
function acceptInt(int $value): void {}
function clean(int $value): int {
    acceptInt(preserve(value: $value));
    acceptInt(evaluate(fn(): int => $value));
    return preserve($value);
}
