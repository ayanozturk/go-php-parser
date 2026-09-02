<?php

/** @param callable(): MissingCallableReturn $factory */
function invokeDocumentedFactory(callable $factory): void
{
    $factory();
}
