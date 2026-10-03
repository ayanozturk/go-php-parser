<?php

$writer = function () use (&$captured): void
{
    $captured = 'ready';
};

echo $captured;
