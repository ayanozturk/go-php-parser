<?php

trait WeakIterableTrait
{
    /** @return array */
    public function values(): array
    {
        return [];
    }

    /** @param array $values */
    public function replace(array $values): void
    {
    }
}

final class TraitBackedValueStore
{
    use WeakIterableTrait;
}
