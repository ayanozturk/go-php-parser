<?php

/**
 * @param array{count: int, label: 'ready'|'busy', nested: array{active: bool}} $data
 */
function inspectShape(array $data): void
{
    acceptInt($data['count']);
    acceptString($data['label']);
    acceptBool($data['nested']['active']);
}

function acceptInt(int $value): void {}
function acceptString(string $value): void {}
function acceptBool(bool $value): void {}
