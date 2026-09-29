<?php

/** @param non-empty-string $text */
function acceptText(string $text): void {}
/** @param non-empty-array<int, int> $values */
function acceptValues(array $values): void {}

function run(): void
{
    $text = 'ready';
    $values = [1];
    acceptText($text);
    acceptValues($values);
}
