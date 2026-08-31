<?php
class TemplateService {}

/**
 * @template T of TemplateService
 * @param class-string<T> $class
 */
function run(string $class): void
{
    $value = new $class();
    $value->missing();
}
