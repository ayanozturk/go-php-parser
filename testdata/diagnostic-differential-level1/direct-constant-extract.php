<?php

function readDirectExtract(): void
{
    extract(['directExtracted' => 1]);
    echo $directExtracted;
}
