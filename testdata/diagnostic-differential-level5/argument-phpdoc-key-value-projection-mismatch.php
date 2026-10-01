<?php
/** @return value-of<array{label: string}> */
function shapeValue(): string { return 'value'; }

/** @return key-of<array{label: string}> */
function shapeKey(): mixed { return 'label'; }

function checkProjection(): void
{
    acceptInt(shapeValue());
    acceptInt(shapeKey());
}

function acceptInt(int $value): void {}
