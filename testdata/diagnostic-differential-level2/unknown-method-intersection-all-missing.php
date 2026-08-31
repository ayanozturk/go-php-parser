<?php
interface FirstContract {}
interface SecondContract {}

function run(FirstContract&SecondContract $value): void
{
    $value->missing();
}
