<?php
class MatchIndexLeft {}
class MatchIndexRight {}

/**
 * @param array{service: callable(): MatchIndexLeft, known: callable(): MatchIndexRight} $factories
 */
function run(array $factories, bool $flag): void
{
    $factories[match ($flag) { true => 'service', false => 'known' }]()->missing();
}
