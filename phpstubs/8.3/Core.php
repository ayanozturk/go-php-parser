<?php

/**
 * Interface to detect if a class is traversable using foreach.
 */
interface Traversable
{
}

/**
 * Interface for external iterators.
 *
 * @template TKey
 * @template TValue
 * @extends Traversable<TKey, TValue>
 */
interface IteratorAggregate extends Traversable
{
    /**
     * @return Traversable<TKey, TValue>
     */
    public function getIterator(): Traversable;
}

/**
 * Interface for external iterators or objects that can be iterated themselves.
 *
 * @template TKey
 * @template TValue
 * @extends Traversable<TKey, TValue>
 */
interface Iterator extends Traversable
{
    public function current(): mixed;

    public function next(): void;

    public function key(): mixed;

    public function valid(): bool;

    public function rewind(): void;
}

interface Throwable
{
    public function getMessage(): string;

    public function getCode(): int;

    public function getFile(): string;

    public function getLine(): int;

    public function getTrace(): array;

    public function getTraceAsString(): string;

    public function getPrevious(): ?Throwable;

    public function __toString(): string;
}

interface Stringable
{
    public function __toString(): string;
}

class Exception implements Throwable, Stringable
{
    public function __construct(string $message = "", int $code = 0, ?Throwable $previous = null) {}

    public function getMessage(): string { return ""; }

    public function getCode(): int { return 0; }

    public function getFile(): string { return ""; }

    public function getLine(): int { return 0; }

    public function getTrace(): array { return []; }

    public function getTraceAsString(): string { return ""; }

    public function getPrevious(): ?Throwable { return null; }

    public function __toString(): string { return ""; }
}

class Error implements Throwable, Stringable
{
    public function __construct(string $message = "", int $code = 0, ?Throwable $previous = null) {}

    public function getMessage(): string { return ""; }

    public function getCode(): int { return 0; }

    public function getFile(): string { return ""; }

    public function getLine(): int { return 0; }

    public function getTrace(): array { return []; }

    public function getTraceAsString(): string { return ""; }

    public function getPrevious(): ?Throwable { return null; }

    public function __toString(): string { return ""; }
}

class ErrorException extends Exception
{
    public function __construct(string $message = "", int $code = 0, int $severity = 1, ?string $filename = null, ?int $line = null, ?Throwable $previous = null) {}

    public function getSeverity(): int { return 0; }
}

class CompileError extends Error {}

class ParseError extends CompileError {}

class TypeError extends Error {}

class ArgumentCountError extends TypeError {}

class ValueError extends Error {}

class ArithmeticError extends Error {}

class DivisionByZeroError extends ArithmeticError {}

class UnhandledMatchError extends Error {}

class stdClass
{
}

final class Closure
{
    private function __construct() {}

    public function __invoke(mixed ...$args): mixed { return null; }

    public function bindTo(?object $newThis, object|string|null $newScope = "static"): ?Closure { return null; }

    public static function bind(Closure $closure, ?object $newThis, object|string|null $newScope = "static"): ?Closure { return null; }

    public function call(object $newThis, mixed ...$args): mixed { return null; }

    public static function fromCallable(callable $callback): Closure { return new self(); }
}

final class Generator implements Iterator
{
    public function current(): mixed { return null; }

    public function key(): mixed { return null; }

    public function next(): void {}

    public function rewind(): void {}

    public function valid(): bool { return false; }

    public function send(mixed $value): mixed { return null; }

    public function throw(Throwable $exception): mixed { return null; }

    public function getReturn(): mixed { return null; }
}

final class WeakReference
{
    public static function create(object $object): WeakReference { return new WeakReference(); }

    public function get(): ?object { return null; }
}

/**
 * @template TKey of object
 * @template TValue
 * @implements ArrayAccess<TKey, TValue>
 * @implements IteratorAggregate<TKey, TValue>
 */
final class WeakMap implements ArrayAccess, Countable, IteratorAggregate
{
    public function offsetExists(mixed $object): bool { return false; }

    public function offsetGet(mixed $object): mixed { return null; }

    public function offsetSet(mixed $object, mixed $value): void {}

    public function offsetUnset(mixed $object): void {}

    public function count(): int { return 0; }

    public function getIterator(): Iterator { return new ArrayIterator([]); }
}

final class Attribute
{
    public const TARGET_CLASS = 1;
    public const TARGET_FUNCTION = 2;
    public const TARGET_METHOD = 4;
    public const TARGET_PROPERTY = 8;
    public const TARGET_CLASS_CONSTANT = 16;
    public const TARGET_PARAMETER = 32;
    public const TARGET_ALL = 63;
    public const IS_REPEATABLE = 128;

    public int $flags;

    public function __construct(int $flags = self::TARGET_ALL) {}
}

interface UnitEnum
{
    /** @return static[] */
    public static function cases(): array;
}

interface BackedEnum extends UnitEnum
{
    /** @return static */
    public static function from(int|string $value): static;

    /** @return static|null */
    public static function tryFrom(int|string $value): ?static;
}

final class Fiber
{
    public function __construct(callable $callback) {}

    public function start(mixed ...$args): mixed { return null; }

    public function resume(mixed $value = null): mixed { return null; }

    public function throw(Throwable $exception): mixed { return null; }

    public function isStarted(): bool { return false; }

    public function isSuspended(): bool { return false; }

    public function isRunning(): bool { return false; }

    public function isTerminated(): bool { return false; }

    public function getReturn(): mixed { return null; }

    public static function getCurrent(): ?self { return null; }

    public static function suspend(mixed $value = null): mixed { return null; }
}

final class FiberError extends Error
{
}

final class ReturnTypeWillChange
{
}

final class AllowDynamicProperties
{
}

final class SensitiveParameter
{
}

final class SensitiveParameterValue
{
    public function __construct(mixed $value) {}

    public function getValue(): mixed { return null; }
}

final class Override
{
}
