<?php
class PropertyCallableService
{
    public function execute(): void {}
}

class PropertyHolder
{
    /** @var callable(): PropertyCallableService */
    public $factory;
}

function run(PropertyHolder $holder): void
{
    $factory = $holder->factory;
    $factory()->execute();
}
