<?php

final class ArrayAccumulatorRecord
{
    /** @var array<string, int> */
    public array $values = [];
}

/** @param array<string, int> $values */
function mergeValues(ArrayAccumulatorRecord $record, array $values): void
{
    $record->values += $values;
}
