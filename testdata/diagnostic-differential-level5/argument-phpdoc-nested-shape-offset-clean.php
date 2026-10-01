<?php
/** @return array{nested: array{active: bool}}['nested']['active'] */
function nestedActive(): bool { return true; }

/** @param array{label?: string} $options */
function optionalLabel(array $options): void
{
    acceptString($options['label']);
}

function checkNestedShape(): void
{
    acceptBool(nestedActive());
}

function acceptBool(bool $value): void {}
function acceptString(string $value): void {}
