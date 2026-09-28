<?php

function returnOnlyOneBranch(bool $stop): string
{
    if ($stop) {
        return 'done';
    }

    return 'continue';
}
