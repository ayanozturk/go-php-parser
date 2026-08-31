<?php
class StringIndexLeft {}
class StringIndexRight {}

/**
 * @param array{left: callable(): StringIndexLeft, right: callable(): StringIndexRight} $factories
 */
function run(array $factories, string $name): void
{
    $factories[$name]()->missing();
}
