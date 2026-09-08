<?php

/**
 * @return array{
 *     label: string,
 *     count: int
 * }|null
 */
function build_payload(bool $available): ?array
{
    if (!$available) {
        return null;
    }

    return ['label' => 'ready', 'count' => 1];
}
