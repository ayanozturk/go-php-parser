<?php

/** @param int<1, 10> $value */
function acceptOneToTen(int $value): void {}

function run(): void
{
    acceptOneToTen(0);
    acceptOneToTen(11);
}
