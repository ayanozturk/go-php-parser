<?php
class GlobalConstShapeService {}

const KEY = 'service';

/** @param array{service: callable(): GlobalConstShapeService} $factories */
function run(array $factories): void
{
    $factories[KEY]()->missing();
}
