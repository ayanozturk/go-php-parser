<?php
class CallableVariableService
{
    public function execute(): void {}
}

/** @param callable(): CallableVariableService $factory */
function run(callable $factory): void
{
    $factory()->execute();
}
