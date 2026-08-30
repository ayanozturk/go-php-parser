<?php
class KnownService
{
    public function execute(): void {}
}

function run(KnownService $service): void
{
    $service->execute();
}
