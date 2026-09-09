<?php

enum Language: string
{
    case Primary = 'primary';

    /** @return array<string> */
    public static function values(): array
    {
        return [self::Primary->value];
    }
}

final class Batch
{
    public function __construct(
        /** @var list<string> */
        public array $items,
        /** @var array<string, mixed> */
        public array $metadata,
    ) {
    }
}
