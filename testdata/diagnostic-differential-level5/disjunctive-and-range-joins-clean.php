<?php

/** @param 'red'|'blue' $color */
function acceptRedOrBlue($color): void {}
/** @param int<1, 10> $value */
function acceptBounded(int $value): void {}

function clean($color, bool $chooseRed, bool $chooseLow): void
{
    if ($color === 'red' || $color === 'blue') {
        acceptRedOrBlue($color);
    }
    if ($chooseRed) { $picked = 'red'; } else { $picked = 'blue'; }
    acceptRedOrBlue($picked);

    if ($chooseLow) { $value = 1; } else { $value = 10; }
    acceptBounded($value);
}
