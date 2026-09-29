<?php

/** @param non-empty-string $value */
function acceptNonEmpty(string $value): void {}

function guarded(?string $value): void
{
    if (is_string($value) && $value !== '') {
        acceptNonEmpty($value);
    }
}
