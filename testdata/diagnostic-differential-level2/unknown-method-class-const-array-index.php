<?php
class ConstShapeService {}

class Holder
{
    public const KEY = 'service';

    /**
     * @param array{service: callable(): ConstShapeService} $factories
     */
    public function run(array $factories): void
    {
        $factories[self::KEY]()->missing();
    }
}
