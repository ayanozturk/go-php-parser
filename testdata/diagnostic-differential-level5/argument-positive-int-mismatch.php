<?php

/** @param positive-int $value */
function acceptPositive(int $value): void {}

function run(): void
{
    acceptPositive(0);
}
