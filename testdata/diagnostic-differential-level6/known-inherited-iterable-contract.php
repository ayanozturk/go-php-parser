<?php

interface TypedCollectionContract
{
    /** @return array<string, int> */
    public function items(): array;

    /** @param array<string, int> $items */
    public function replace(array $items): void;

    /** @param array<string, int> $items */
    public function append(array $items): void;
}

final class InheritedCollection implements TypedCollectionContract
{
    public function items(): array
    {
        return [];
    }

    public function replace(array $items): void
    {
    }

    public function append($items): void
    {
    }
}
