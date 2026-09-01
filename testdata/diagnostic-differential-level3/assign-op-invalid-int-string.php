<?php

final class LevelThreeCounter
{
    public int $count = 0;
}

function addTextToLevelThreeCount(LevelThreeCounter $counter): void
{
    $counter->count += 'text';
}
