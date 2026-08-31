<?php
class AssignedShapeService
{
    public function execute(): void {}
}

/** @param array{service: callable(): AssignedShapeService} $factories */
function run(array $factories): void
{
    $key = "service";
    $factories[$key]()->execute();
}
