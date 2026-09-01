<?php

final class InstanceCounter
{
    public int $count = 0;

    public static function replaceCount(): void
    {
        self::$count = 'text';
    }
}
