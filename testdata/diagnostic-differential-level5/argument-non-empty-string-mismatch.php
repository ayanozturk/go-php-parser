<?php

/** @param non-empty-string $value */
function acceptNonEmptyString($value): void {}

function run(): void
{
    acceptNonEmptyString('');
}

/** @param array{blank: ''} $data */
function passEmptyShapeLiteral(array $data): void
{
    acceptNonEmptyString($data['blank']);
}
