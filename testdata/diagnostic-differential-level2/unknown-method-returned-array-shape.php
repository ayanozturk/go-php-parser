<?php
class ReturnedShapeService {}

class Holder
{
    /**
     * @return array{service: callable(): ReturnedShapeService}
     */
    public function factories(): array
    {
        return ['service' => static fn (): ReturnedShapeService => new ReturnedShapeService()];
    }
}

function run(Holder $holder): void
{
    $holder->factories()["service"]()->missing();
}
