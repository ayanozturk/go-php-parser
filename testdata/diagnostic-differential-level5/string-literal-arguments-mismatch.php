<?php

/** @param 'red'|'blue' $color */
function acceptColor($color): void {}
/** @param key-of<array{red: int, blue: int}> $key */
function acceptColorKey($key): void {}

function invalid(): void
{
    $color = 'green';
    acceptColor('green');
    acceptColor($color);
    acceptColorKey('green');
}
