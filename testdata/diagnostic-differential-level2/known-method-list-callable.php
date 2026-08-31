<?php
class ListCallableService
{
    public function execute(): void {}
}

/** @param list{callable(): ListCallableService} $factories */
function run(array $factories): void
{
    $factories[0]()->execute();
}
