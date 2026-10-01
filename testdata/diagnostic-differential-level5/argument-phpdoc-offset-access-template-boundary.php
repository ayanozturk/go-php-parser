<?php
/**
 * @template T of array<array-key, mixed>
 * @template K of key-of<T>
 * @param T $values
 * @param K $key
 * @return T[K]
 */
function getOffset(array $values, $key)
{
    return $values[$key];
}
