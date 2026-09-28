<?php

/** @template Tv */
class GenericSearch
{
    /** @param Tv $value */
    public function search($value): void {}
}

function run(GenericSearch $search): void
{
    $search->search(1);
}
