<?php
class StringIndexLeft {}
class StringIndexRight
{
    public function execute(): void {}
}

/**
 * @param array{left: callable(): StringIndexLeft, right: callable(): StringIndexRight} $factories
 */
function run(array $factories, string $name): void
{
    $factories[$name]()->execute();
}
