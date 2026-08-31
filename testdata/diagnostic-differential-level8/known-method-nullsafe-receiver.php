<?php
class NullsafeService
{
    public function execute(): void {}
}

function run(?NullsafeService $value): void
{
    $value?->execute();
}
