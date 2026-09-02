<?php

/** @param array<int, MissingNestedService> $items */
function inspectNestedUnknownClass(array $items): void
{
    echo count($items);
}
