<?php
class TemplateService
{
    public function execute(): void {}
}

/**
 * @template T of TemplateService
 * @param class-string<T> $class
 */
function run(string $class): void
{
    $value = new $class();
    $value->execute();
}
