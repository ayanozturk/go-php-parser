<?php
class ConcatShapeService {}

/** @param array{service: callable(): ConcatShapeService} $factories */
function run(array $factories): void
{
    $factories["serv" . "ice"]()->missing();
}
