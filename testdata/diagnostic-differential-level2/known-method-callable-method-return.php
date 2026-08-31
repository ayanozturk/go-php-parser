<?php
class MethodCallableService
{
    public function execute(): void {}
}

class CallableMaker
{
    /** @return callable(): MethodCallableService */
    public function make(): callable
    {
        return static fn (): MethodCallableService => new MethodCallableService();
    }
}

(new CallableMaker())->make()()->execute();
