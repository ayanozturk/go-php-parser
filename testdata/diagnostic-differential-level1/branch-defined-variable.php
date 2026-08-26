<?php

function printOptionalValue(bool $shouldPrint): void
{
    if ($shouldPrint) {
        $value = 'ready';
    }

    echo $value;
}
