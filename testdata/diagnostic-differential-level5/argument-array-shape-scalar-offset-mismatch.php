<?php

/** @param array{count: int} $data */
function inspectShape(array $data): void
{
    acceptString($data['count']);
}

function acceptString(string $value): void {}
