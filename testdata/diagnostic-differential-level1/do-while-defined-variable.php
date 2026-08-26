<?php

function printDoWhileValue(bool $shouldContinue): void
{
    do {
        $value = 'ready';
    } while ($shouldContinue);

    echo $value;
}
