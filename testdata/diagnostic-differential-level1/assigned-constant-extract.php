<?php

function readAssignedExtract(): void
{
    $assignedValues = ['assignedExtracted' => 1];
    extract($assignedValues);
    echo $assignedExtracted;
}
