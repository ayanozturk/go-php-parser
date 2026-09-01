<?php

final class IntegerCounter
{
    public int $count = 0;
}

function incrementCount(IntegerCounter $counter): void
{
    $counter->count += 1;
}
