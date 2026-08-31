<?php
class ShapeCallableService {}

/** @param array{service: callable(): ShapeCallableService} $factories */
function run(array $factories): void
{
    $factory = $factories["service"];
    $factory()->missing();
}
