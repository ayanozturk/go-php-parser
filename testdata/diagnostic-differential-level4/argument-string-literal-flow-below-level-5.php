<?php

/** @param 'ready' $value */
function acceptReadyLiteral($value): void {}

/** @param 'ready'|'busy' $value */
function acceptReadyOrBusyLiteral($value): void {}

function checkStringLiteralAssignmentsBelowLevel5(bool $flag): void
{
    $ready = 'ready';
    acceptReadyLiteral($ready);

    $busy = 'busy';
    acceptReadyLiteral($busy);

    if ($flag) {
        $joined = 'ready';
    } else {
        $joined = 'busy';
    }
    acceptReadyOrBusyLiteral($joined);
}

/** @param 'ready'|'busy' $value */
function checkStringLiteralNarrowingBelowLevel5($value): void
{
    if ($value !== 'busy') {
        acceptReadyLiteral($value);
    }
    if ($value === 'busy') {
        acceptReadyLiteral($value);
    }
}
