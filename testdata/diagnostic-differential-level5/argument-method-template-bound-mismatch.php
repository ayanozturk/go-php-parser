<?php

class StringSearch
{
    /**
     * @template Tin of string
     * @param Tin $value
     */
    public function search($value): void {}
}

function run(StringSearch $search): void
{
    $search->search(1);
}
