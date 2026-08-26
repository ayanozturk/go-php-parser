<?php

function readAfterNestedBreak(): void
{
    do {
        do {
            $value = 'ready';
            break 2;
        } while (true);
    } while (true);

    echo $value;
}
