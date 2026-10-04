<?php

final class InitializedReadonlyProperty
{
    public readonly int $id;
    public readonly int $branch;

    public function __construct(bool $left)
    {
        $this->id = 1;
        if ($left) {
            $this->branch = 1;
        } else {
            $this->branch = 2;
        }
    }
}
