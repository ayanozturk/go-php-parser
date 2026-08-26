<?php

function readAfterBreakThroughFinally(): void
{
    do {
        do {
            try {
                break 2;
            } finally {
                $inFinally = 'ready';
            }
        } while (true);

        $afterFinally = 'not-reached';
    } while (true);

    echo $afterFinally;
}
