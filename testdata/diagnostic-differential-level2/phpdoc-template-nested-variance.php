<?php

/** @template-covariant T */
interface ReadBox {}

/** @template-contravariant T */
interface WriteBox {}

/** @template T */
interface InvariantBox {}

/** @template-covariant T */
final class NestedVariance
{
    /** @return ReadBox<T> */
    public function readThroughCovariant() {}

    /** @param ReadBox<T> $value */
    public function invalidWriteThroughCovariant($value): void {}

    /** @return WriteBox<T> */
    public function invalidReadThroughContravariant() {}

    /** @param WriteBox<T> $value */
    public function writeThroughContravariant($value): void {}

    /** @return InvariantBox<T> */
    public function invalidInvariantReturn() {}

    /** @return callable(T): void */
    public function invalidCallableInput() {}

    /** @param callable(T): void $callback */
    public function acceptCallableInput($callback): void {}

    /** @param callable(): T $callback */
    public function invalidCallableOutput($callback): void {}

    /** @return callable(): T */
    public function callableOutput() {}
}
