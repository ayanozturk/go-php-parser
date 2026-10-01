<?php

/**
 * @param bool $asInt
 * @return ($asInt is true ? int : string)
 */
function conditionalReturnDoesNotFitNative($asInt): int
{
    return 1;
}
