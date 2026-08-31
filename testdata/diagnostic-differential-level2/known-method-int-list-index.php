<?php
class IntListService
{
    public function execute(): void {}
}

/** @param list{callable(): IntListService} $list */
function run(array $list, int $i): void
{
    $list[$i]()->execute();
}
