<?php

/** @param non-empty-array<int, int> $values */
function acceptNonEmptyArray(array $values): void {}

/** @param array<int, int> $values */
function passUnknownArray(array $values): void
{
    acceptNonEmptyArray($values);
}
