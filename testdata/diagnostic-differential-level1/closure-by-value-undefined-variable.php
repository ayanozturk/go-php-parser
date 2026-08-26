<?php

$printer = function () use ($missing): void
{
    echo $missing;
};

$printer();
