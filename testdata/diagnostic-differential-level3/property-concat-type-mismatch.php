<?php

final class NumericLabel
{
    public int $value = 0;
}

function appendLabel(NumericLabel $label): void
{
    $label->value .= 'suffix';
}
