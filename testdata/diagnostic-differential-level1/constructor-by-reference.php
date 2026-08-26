<?php

class ConstructorWriter
{
    public function __construct(&$out)
    {
        $out = 'ready';
    }
}

function printConstructedValue(): void
{
    new ConstructorWriter($value);
    echo $value;
}
