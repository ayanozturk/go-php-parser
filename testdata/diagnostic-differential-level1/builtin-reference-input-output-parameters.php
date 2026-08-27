<?php

function readBuiltinInputs(): void
{
    array_splice($missingItems, 0, 1);
    settype($missingValue, 'string');
    sort($missingList);
}
