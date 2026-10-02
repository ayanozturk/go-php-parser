<?php

class LiteralState
{
    /** @var 'ready' */
    public string $state;

    /** @return 'ready' */
    public function ready(): string
    {
        return 'ready';
    }

    /** @return 'ready' */
    public function invalid(): string
    {
        return 'wrong';
    }
}

/** @param 'ready' $value */
function acceptReadyValue($value): void {}

function runLiteralReturnAndPropertyChecks(LiteralState $state): void
{
    acceptReadyValue($state->ready());
    acceptReadyValue($state->state);
    acceptReadyValue('wrong');
    $state->state = 'wrong';
}
