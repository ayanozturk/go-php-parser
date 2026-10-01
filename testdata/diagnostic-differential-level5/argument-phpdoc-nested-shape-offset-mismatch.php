<?php
/** @return array{nested: array{active: bool}}['nested']['active'] */
function nestedActive(): bool { return true; }

function checkNestedShape(): void
{
    acceptString(nestedActive());
}

function acceptString(string $value): void {}
