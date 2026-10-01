<?php

/** @param 'red'|'blue' $color */
function acceptColor($color): void {}
/** @param 'red' $color */
function acceptRed($color): void {}
/** @param 'blue' $color */
function acceptBlue($color): void {}

/** @param 'red'|'blue' $color */
function check($color, bool $enabled): void
{
    if ($color === 'red') { acceptRed($color); } else { acceptBlue($color); }
    if ($color !== 'red') { acceptBlue($color); } else { acceptRed($color); }
    if ($enabled && $color === 'red') { acceptRed($color); }
}

function branchJoin(bool $useRed): void
{
    if ($useRed) { $color = 'red'; } else { $color = 'blue'; }
    acceptColor($color);
}
