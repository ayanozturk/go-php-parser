<?php

/** @param 'red'|'blue' $color */
function acceptColor($color): void {}
/** @param key-of<array{red: int, blue: int}> $key */
function acceptColorKey($key): void {}

function valid(): void
{
    $color = 'red';
    acceptColor('blue');
    acceptColor($color);
    acceptColorKey('red');
}
