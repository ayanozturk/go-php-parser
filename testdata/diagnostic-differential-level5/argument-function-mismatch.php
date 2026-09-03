<?php

function acceptInt(int $value): void {}

function runFunctionMismatch(): void
{
    acceptInt('wrong');
}
