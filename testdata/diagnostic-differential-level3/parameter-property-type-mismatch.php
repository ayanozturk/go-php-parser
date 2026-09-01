<?php

final class NumberRecord
{
    public string $label;
}

function assignNumberToLabel(NumberRecord $record): void
{
    $record->label = 42;
}
