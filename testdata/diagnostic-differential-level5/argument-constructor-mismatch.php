<?php

final class ConstructorTarget
{
    public function __construct(int $value)
    {
        echo $value;
    }
}

function runConstructorMismatch(): void
{
    $target = new ConstructorTarget('wrong');
    echo get_class($target);
}
