<?php

function stopsExecution(): never
{
    throw new RuntimeException('stop');
}
