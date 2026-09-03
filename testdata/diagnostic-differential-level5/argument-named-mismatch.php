<?php

function acceptNamed(int $count, string $label): void {}

function runNamedMismatch(): void
{
    acceptNamed(label: 'ok', count: 'wrong');
}
