<?php

interface IterableContract
{
    /** @return array<string, int> */
    public function values(): array;

    /** @param array<string, int> $values */
    public function replace(array $values): void;
}

function makeIterableContract(): object
{
    return new class implements IterableContract {
        /** @return array */
        public function values(): array
        {
            return [];
        }

        /** @param array $values */
        public function replace(array $values): void
        {
        }
    };
}
