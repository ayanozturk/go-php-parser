<?php

function printWhileValue(bool $shouldRun): void
{
    while ($shouldRun) {
        $value = 'ready';
        $shouldRun = false;
    }

    echo $value;
}
