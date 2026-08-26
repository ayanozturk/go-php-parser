<?php

function readKnownDynamicName(): void
{
    $knownName = 'missingDynamicTarget';
    echo $$knownName;
}
