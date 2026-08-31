<?php
interface ExecutableContract
{
    public function execute(): void;
}

interface MarkerContract {}

function run((ExecutableContract&MarkerContract)|null $value): void
{
    $value->execute();
}
