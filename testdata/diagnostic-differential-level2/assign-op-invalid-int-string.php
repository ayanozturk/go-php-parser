<?php

final class AdditiveCounter
{
    public int $count = 0;
}

function addTextToCount(AdditiveCounter $counter): void
{
    $counter->count += 'text';
}
