<?php

final class CleanTarget
{
    public function __construct(int $value)
    {
        echo $value;
    }

    public function accept(string $value): void {}
}

function acceptString(string $value): void {}

function runClean(CleanTarget $target): void
{
    acceptString('ok');
    $target->accept('ok');
    $created = new CleanTarget(1);
    echo get_class($created);
}
