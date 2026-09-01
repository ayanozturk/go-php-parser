<?php

final class StringLabel
{
    public string $value = '';
}

function appendNumber(StringLabel $label): void
{
    $label->value .= 1;
}
