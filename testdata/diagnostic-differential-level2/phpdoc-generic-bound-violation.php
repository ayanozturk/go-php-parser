<?php

class BoundRoot
{
}

class UnrelatedValue
{
}

/** @template TValue of BoundRoot */
final class BoundedContainer
{
}

/** @param BoundedContainer<UnrelatedValue> $container */
function inspectBoundedContainer(BoundedContainer $container): void
{
    echo 'checked';
}
