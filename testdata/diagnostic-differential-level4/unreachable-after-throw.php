<?php

function throwBeforeDeadWork(): never
{
    throw new RuntimeException('stop');

    echo 'unreachable';
}
