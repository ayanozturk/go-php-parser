<?php

interface PreciseCollectionContract
{
    /** @return array<string, int> */
    public function items(): array;

    /** @param array<string, int> $items */
    public function replace(array $items): void;
}

final class WeakCollection implements PreciseCollectionContract
{
    /** @return array */
    public function items(): array
    {
        return [];
    }

    /** @param array $items */
    public function replace(array $items): void
    {
    }
}
