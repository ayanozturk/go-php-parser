<?php
class ForeignConstShapeService {}

class Other
{
    public const KEY = 'service';
}

/** @param array{service: callable(): ForeignConstShapeService} $factories */
function run(array $factories): void
{
    $factories[Other::KEY]()->missing();
}
