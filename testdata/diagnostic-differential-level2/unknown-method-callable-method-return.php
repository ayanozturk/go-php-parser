<?php
class MethodCallableService {}

class CallableMaker
{
    /** @return callable(): MethodCallableService */
    public function make(): callable
    {
        return static fn (): MethodCallableService => new MethodCallableService();
    }
}

(new CallableMaker())->make()()->missing();
