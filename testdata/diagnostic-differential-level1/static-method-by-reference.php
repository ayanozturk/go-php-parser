<?php

class StaticWriter
{
    public static function fill(&$out): void
    {
        $out = 'ready';
    }

    public static function run(): void
    {
        self::fill($selfValue);
        StaticWriter::fill($explicitValue);
        echo $selfValue;
        echo $explicitValue;
    }
}
