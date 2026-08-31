<?php
class DynamicClassService
{
    public function execute(): void {}
}

/** @param class-string<DynamicClassService> $class */
function run(string $class): void
{
    $value = new $class();
    $value->execute();
}
