<?php

final class KnownService
{
}

/** @template TValue */
final class GenericContainer
{
}

/** @param GenericContainer<KnownService> $container */
function inspectGenericKnownArgument(GenericContainer $container): void
{
    echo 'checked';
}
