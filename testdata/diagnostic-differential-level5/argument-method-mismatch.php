<?php

final class MethodTarget
{
    public function accept(int $value): void {}
}

function runMethodMismatch(MethodTarget $target): void
{
    $target->accept('wrong');
}
