<?php

final class ConditionalReadonlyProperty
{
    public readonly int $id;

    public function __construct(bool $initialize)
    {
        if ($initialize) {
            $this->id = 1;
        }
    }
}
