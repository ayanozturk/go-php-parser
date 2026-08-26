<?php

function readAfterNestedContinue(): void
{
    do {
        do {
            $before = 'ready';
            continue 2;
        } while (false);

        $after = 'not-reached';
    } while (false);

    echo $after;
}
