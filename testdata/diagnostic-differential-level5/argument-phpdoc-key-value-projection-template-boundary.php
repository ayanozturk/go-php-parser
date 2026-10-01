<?php
interface GenericArrayProjection
{
    /**
     * @template T of array<array-key, mixed>
     * @param T $values
     * @return value-of<T>
     */
    public function value(array $values): mixed;

    /**
     * @template T of array<array-key, mixed>
     * @param T $values
     * @return key-of<T>
     */
    public function key(array $values): int|string;
}
