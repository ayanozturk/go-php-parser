<?php
class PropertyShapeService
{
    public function execute(): void {}
}

class Holder
{
    /** @var array{service: callable(): PropertyShapeService} */
    public array $factories;
}

function run(Holder $holder): void
{
    $holder->factories["service"]()->execute();
}
