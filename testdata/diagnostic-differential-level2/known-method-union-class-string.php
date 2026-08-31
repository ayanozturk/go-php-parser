<?php
class KnownUnionStringService
{
    public function execute(): void {}
}

function run(KnownUnionStringService|string $value): void
{
    $value->execute();
}
