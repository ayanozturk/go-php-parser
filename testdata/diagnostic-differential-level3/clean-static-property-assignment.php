<?php

final class StaticCounter
{
    public static int $count = 0;

    public static function increment(): void
    {
        self::$count = 1;
    }
}
