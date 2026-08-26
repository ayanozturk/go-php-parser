<?php

function readAfterSwitchContinue(int $value): void
{
    do {
        switch ($value) {
            default:
                continue 2;
        }

        $afterSwitch = 'not-reached';
    } while (false);

    echo $afterSwitch;
}
