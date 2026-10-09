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
function wrong(): int {
    acceptInt(preserve('wrong'));
    acceptInt(evaluate(fn(): string => 'wrong'));
    return preserve('wrong');
}
