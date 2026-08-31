<?php
class ShapeCallableService
{
    public function execute(): void {}
}

/** @param array{service: callable(): ShapeCallableService} $factories */
function run(array $factories): void
{
    $factory = $factories["service"];
    $factory()->execute();
}
