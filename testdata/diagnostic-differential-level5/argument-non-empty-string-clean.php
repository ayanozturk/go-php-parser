<?php

/** @param non-empty-string $value */
function acceptNonEmptyString($value): void {}

function run(): void
{
    acceptNonEmptyString('ready');
}

/** @param array{label: 'ready'} $data */
function passNonEmptyShapeLiteral(array $data): void
{
    acceptNonEmptyString($data['label']);
}
