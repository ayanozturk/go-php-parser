<?php

interface EventMapContract
{
    /** @return array<string, string|array{0: string, 1: int}|list<array{0: string, 1?: int}>> */
    public static function events();

    /** @return EventMap[] */
    public function maps();
}

final class EventMap implements EventMapContract
{
    public static function events(): array
    {
        return [];
    }

    public function maps(): array
    {
        return [];
    }
}
