<?php

class Collection
{
    public function filter(Closure $p): void
    {
    }
}

function run(Collection $items): void
{
    $items->filter(static fn ($item) => true);
}
