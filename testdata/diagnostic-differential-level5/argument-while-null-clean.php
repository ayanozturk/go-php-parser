<?php

class Task
{
    public function parent(): ?Task
    {
        return null;
    }
}

function sumChildren(Task $task): void
{
}

function run(Task $task): void
{
    $current = $task->parent();
    $depth = 0;
    while ($current !== null && $depth < 64) {
        sumChildren($current);
        $current = $current->parent();
        $depth++;
    }
}
