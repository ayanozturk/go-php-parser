<?php

final class ArrayAccumulator
{
    /** @var array<string, int> */
    public array $values = [];
}

function addNumberToValues(ArrayAccumulator $accumulator): void
{
    $accumulator->values += 1;
}
