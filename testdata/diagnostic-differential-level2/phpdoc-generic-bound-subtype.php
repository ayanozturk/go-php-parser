<?php

class BoundRoot
{
}

final class BoundChild extends BoundRoot
{
}

/** @template TValue of BoundRoot */
final class BoundedContainer
{
}

/** @param BoundedContainer<BoundChild> $container */
function inspectBoundedContainer(BoundedContainer $container): void
{
    echo 'checked';
}
