<?php

function reportInvalidArrayOperation(array $items, int $offset): void
{
    $result = $items + $offset;
    echo 'checked';
}
