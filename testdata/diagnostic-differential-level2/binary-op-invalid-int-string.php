<?php

function reportInvalidNumericOperation(int $number, string $text): void
{
    $result = $number + $text;
    echo 'checked';
}
