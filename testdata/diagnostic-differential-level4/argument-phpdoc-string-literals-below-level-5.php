<?php

/** @param 'Ready' $value */
function acceptReadyLiteral($value): void {}

/** @param 'red'|'blue' $value */
function acceptColorLiteral($value): void {}

/** @param 'Ready' $value */
function acceptReadyFromString(string $value): void {}

function acceptGeneralString(string $value): void {}

/** @param value-of<array{state: 'Ready'|'Busy'}> $value */
function acceptProjectedState($value): void {}

function runPhpDocStringLiteralArgumentsBelowLevel5(string $value): void
{
    acceptReadyLiteral('Ready');
    acceptReadyLiteral('ready');
    acceptColorLiteral('red');
    acceptColorLiteral('green');
    acceptReadyFromString('Ready');
    acceptReadyFromString($value);
    acceptGeneralString('arbitrary');
    acceptGeneralString(42);
    acceptProjectedState('Ready');
    acceptProjectedState('Busy');
    acceptProjectedState('Done');
}
