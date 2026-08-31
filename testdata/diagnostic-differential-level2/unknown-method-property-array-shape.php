<?php
class PropertyShapeService {}

class Holder
{
    /** @var array{service: callable(): PropertyShapeService} */
    public array $factories;
}

function run(Holder $holder): void
{
    $holder->factories["service"]()->missing();
}
