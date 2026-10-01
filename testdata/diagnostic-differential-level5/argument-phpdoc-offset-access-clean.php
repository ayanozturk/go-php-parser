<?php
/**
 * @phpstan-type Summary array{count: int, label: string}
 */
class SummaryProvider
{
    /** @return Summary['count'] */
    public function count(): int
    {
        return 1;
    }
}

/** @param array<int, string> $items */
function inspectOffsets(array $items, SummaryProvider $provider): void
{
    acceptInt($provider->count());
    acceptString($items[0]);
}

function acceptInt(int $value): void {}
function acceptString(string $value): void {}
