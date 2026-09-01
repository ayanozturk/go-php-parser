<?php

function conditionallyReturnsInteger(bool $condition): int
{
    if ($condition) {
        return 1;
    }
}
