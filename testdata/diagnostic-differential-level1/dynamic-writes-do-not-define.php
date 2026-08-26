<?php

function writeDynamicNames(): void
{
    $writeName = 'dynamicWriteTarget';
    $$writeName = 1;
    echo $dynamicWriteTarget;

    ${'literalDynamicWrite'} = 1;
    echo $literalDynamicWrite;
}
