<?php

$printer = function () use (&$captured): void
{
    $captured = 'ready';
};

$printer();
echo $captured;
