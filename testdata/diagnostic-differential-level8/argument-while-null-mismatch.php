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
    sumChildren($task->parent());
}
