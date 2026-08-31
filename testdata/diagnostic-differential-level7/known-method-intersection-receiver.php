<?php
interface ExecutableContract
{
    public function execute(): void;
}

interface TaggedContract {}

function run(ExecutableContract&TaggedContract $value): void
{
    $value->execute();
}
