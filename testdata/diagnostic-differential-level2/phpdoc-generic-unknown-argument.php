<?php

/** @template TValue */
final class GenericContainer
{
}

/** @param GenericContainer<MissingGenericService> $container */
function inspectGenericUnknownArgument(GenericContainer $container): void
{
    echo 'checked';
}
