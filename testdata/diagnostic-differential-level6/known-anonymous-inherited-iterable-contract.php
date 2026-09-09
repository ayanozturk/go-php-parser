<?php

interface BagContract
{
    /** @return string[] */
    public function values();

    /** @param array<string, mixed> $options */
    public function configure(array $options): void;
}

function createBag(): object
{
    return new class implements BagContract {
        public function values(): array
        {
            return [];
        }

        public function configure(array $options): void
        {
        }
    };
}
