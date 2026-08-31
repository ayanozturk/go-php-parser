<?php
class ListCallableService {}

/** @param list{callable(): ListCallableService} $factories */
function run(array $factories): void
{
    $factories[0]()->missing();
}
