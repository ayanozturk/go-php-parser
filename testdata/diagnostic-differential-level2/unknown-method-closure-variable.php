<?php
class ClosureVariableService {}

function run(): void
{
    $factory = static function (): ClosureVariableService {
        return new ClosureVariableService();
    };
    $factory()->missing();
}
