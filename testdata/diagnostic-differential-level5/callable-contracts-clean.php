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
consumeClosure(fn(int $value): string => 'ready');
consumeCallable(fn(mixed $value): string => 'ready');
consumeCallable(fn(): string => 'ignored');
noInputs(fn(int $value = 1): int => $value);
optionalInput(fn(int $value = 1): int => $value);
twoInputs(fn(int ...$values): int => 1);
ignoreResult(function (int $value): void {});
/** @param callable(int):bool|bool $callback */
function optionalCallback($callback): void {}
optionalCallback(true);
optionalCallback(fn(int $value): bool => true);
/** @param (\Closure():void)|null $callback */
function nullable(?\Closure $callback): \Closure { return $callback ?? function () {}; }
/**
 * @template Key of array-key
 * @param Key $key
 * @param callable(Key):int $convert
 */
function convert($key, callable $convert): int { return $convert($key); }
