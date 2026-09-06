<?php

/**
 * @template TInput of string|int|float|bool|null
 */
final class InputBag
{
    /**
     * @template TDefault of string|int|float|bool|null
     *
     * @param TDefault $default
     *
     * @return TDefault|TInput
     */
    public function get(string $key, mixed $default = null): string|int|float|bool|null
    {
        return $default;
    }
}

function useDate(InputBag $query): void
{
    $date = $query->get('date');
    if (is_string($date)) {
        echo (new DateTime($date))->format('c');
    }
}
