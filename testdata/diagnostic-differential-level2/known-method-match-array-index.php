<?php
class MatchIndexLeft {}
class MatchIndexRight
{
    public function execute(): void {}
}

/**
 * @param array{service: callable(): MatchIndexLeft, known: callable(): MatchIndexRight} $factories
 */
function run(array $factories, bool $flag): void
{
    $factories[match ($flag) { true => 'service', false => 'known' }]()->execute();
}
