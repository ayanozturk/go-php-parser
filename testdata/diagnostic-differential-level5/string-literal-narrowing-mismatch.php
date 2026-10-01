<?php

/** @param 'red'|'blue' $color */
function acceptColor($color): void {}
/** @param 'red' $color */
function acceptRed($color): void {}
/** @param 'blue' $color */
function acceptBlue($color): void {}

/** @param 'red'|'blue' $color */
function invalidBranches($color): void
{
    if ($color === 'red') { acceptBlue($color); } else { acceptRed($color); }
}
