<?php

/**
 * @param bool $asInt
 * @return ($asInt is true ? int : string)
 */
function conditionalReturnMatchesNativeUnion($asInt): int|string
{
    return $asInt ? 1 : 'text';
}
