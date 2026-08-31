<?php
class DynamicClassService {}

/** @param class-string<DynamicClassService> $class */
function run(string $class): void
{
    $value = new $class();
    $value->missing();
}
