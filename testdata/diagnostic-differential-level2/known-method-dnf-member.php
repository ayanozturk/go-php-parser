<?php
interface AvailableContract
{
    public function execute(): void;
}

interface MarkerContract {}
class AlternativeChoice {}

function run((AvailableContract&MarkerContract)|AlternativeChoice $value): void
{
    $value->execute();
}
