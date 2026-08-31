<?php
class ClosureVariableService
{
    public function execute(): void {}
}

function run(): void
{
    $factory = static function (): ClosureVariableService {
        return new ClosureVariableService();
    };
    $factory()->execute();
}
