<?php

class GenericSearch
{
    /**
     * @template Tin
     * @param Tin $value
     */
    public function search($value): void {}
}

function run(GenericSearch $search): void
{
    $search->search(1);
}
