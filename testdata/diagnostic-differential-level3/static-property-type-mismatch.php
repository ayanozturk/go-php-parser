<?php

final class StaticCounter
{
    public static int $count = 0;

    public static function replaceCount(): void
    {
        self::$count = 'text';
    }
}
