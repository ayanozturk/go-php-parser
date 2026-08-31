<?php
class NestedShapeService {}

/** @param array{inner: array{service: callable(): NestedShapeService}} $factories */
function run(array $factories): void
{
    $factory = $factories["inner"]["service"];
    $factory()->missing();
}
