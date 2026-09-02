<?php

/** @template TValue */
final class KnownTypeBox
{
}

/** @param KnownTypeBox<int> $box */
function inspectKnownBox(KnownTypeBox $box): void
{
    echo 'checked';
}
