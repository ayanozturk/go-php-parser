<?php

/**
 * @param bool $notFalse
 * @return ($notFalse is not false ? int : string)
 */
function conditionalNegatedReturnFitsNativeUnion($notFalse): int|string
{
    return $notFalse ? 1 : 'text';
}
