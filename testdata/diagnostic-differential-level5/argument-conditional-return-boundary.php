<?php

function acceptConditionalUnion(int|string $value): void {}

/**
 * @param bool $asInt
 * @return ($asInt is true ? int : string)
 */
function conditionalValue($asInt)
{
    if ($asInt) {
        return 1;
    }
    return 'text';
}

function checkUnresolvedConditional(bool $unknown): void
{
    acceptConditionalUnion(conditionalValue($unknown));
}
