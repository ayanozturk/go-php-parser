<?php
namespace Contracts;
/** @param \Closure(int): string $callback */
function consumeClosure(\Closure $callback): void {}
/** @param callable(int): string $callback */
function consumeCallable(callable $callback): void {}
/** @param callable(): int $callback */
function noInputs(callable $callback): void {}
/** @param callable(int=): int $callback */
function optionalInput(callable $callback): void {}
/** @param callable(int, int): int $callback */
function twoInputs(callable $callback): void {}
/** @param callable(int): void $callback */
function ignoreResult(callable $callback): void {}
consumeClosure(fn(int $value): int => $value);
consumeCallable(fn(string $value): string => $value);
noInputs(fn(int $value): int => $value);
optionalInput(fn(int $value): int => $value);
twoInputs(fn(string ...$values): int => 1);
ignoreResult(fn(): string => 'discarded');
