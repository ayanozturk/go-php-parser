<?php

/** @param int<1, 10> $value */
function acceptOneToTen(int $value): void {}
/** @param int<min, max> $value */
function acceptAnyInt(int $value): void {}

function run(): void
{
    acceptOneToTen(1);
    acceptOneToTen(10);
    acceptAnyInt(0);
}
