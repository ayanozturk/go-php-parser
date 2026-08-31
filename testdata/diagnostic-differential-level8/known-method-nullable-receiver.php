<?php
class NullableService
{
    public function execute(): void {}
}

function run(?NullableService $value): void
{
    $value->execute();
}
