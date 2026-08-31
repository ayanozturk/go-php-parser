<?php
class AssignedShapeService {}

/** @param array{service: callable(): AssignedShapeService} $factories */
function run(array $factories): void
{
    $key = "service";
    $factories[$key]()->missing();
}
