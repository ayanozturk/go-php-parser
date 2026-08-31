<?php
class PropertyCallableService {}

class PropertyHolder
{
    /** @var callable(): PropertyCallableService */
    public $factory;
}

function run(PropertyHolder $holder): void
{
    $factory = $holder->factory;
    $factory()->missing();
}
