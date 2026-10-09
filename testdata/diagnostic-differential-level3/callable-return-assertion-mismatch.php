<?php
namespace Returns;
/**
 * @template Result
 * @param callable(): Result $factory
 * @return Result
 */
function evaluate(callable $factory) { return $factory(); }
function asserted(array|false $value): array {
    /** @var list<string> */
    return evaluate(fn() => $value);
}
function named(array|false $value): array {
    /** @var array $value */
    return $value;
}
function notAsserted(string $value): array {
    return $value;
}
function wrongAssertion(mixed $value): int {
    /** @var string $value */
    return $value;
}
