<?php
class CallableVariableService {}

/** @param callable(): CallableVariableService $factory */
function run(callable $factory): void
{
    $factory()->missing();
}
