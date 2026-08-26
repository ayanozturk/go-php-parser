<?php

function readUnknownExtract(array $unknownValues): void
{
    extract($unknownValues);
    echo $possibleUnknownExtract;
}
