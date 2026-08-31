<?php
class NestedShapeService
{
    public function execute(): void {}
}

/** @param array{inner: array{service: callable(): NestedShapeService}} $factories */
function run(array $factories): void
{
    $factories["inner"]["service"]()->execute();
}
