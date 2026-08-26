<?php

function printBranchValue(bool $enabled): void
{
    if ($enabled) {
        $value = 'enabled';
    } else {
        $value = 'disabled';
    }

    echo $value;
}
