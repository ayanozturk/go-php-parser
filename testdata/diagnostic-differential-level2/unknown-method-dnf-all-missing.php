<?php
interface FirstContract {}
interface SecondContract {}
class AlternativeChoice {}

function run((FirstContract&SecondContract)|AlternativeChoice $value): void
{
    $value->missing();
}
