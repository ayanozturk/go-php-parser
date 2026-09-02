<?php

/** @param callable(MissingCallableParameter): void $handler */
function invokeDocumentedHandler(callable $handler): void
{
    $handler(new stdClass());
}
