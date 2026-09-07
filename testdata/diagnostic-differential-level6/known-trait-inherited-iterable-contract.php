<?php

trait TypedCollectionTrait
{
    /** @return array<string, int> */
    public function items(): array
    {
        return [];
    }

    /** @param array<string, int> $items */
    public function replace(array $items): void
    {
    }
}

final class TraitBackedCollection
{
    use TypedCollectionTrait;
}
