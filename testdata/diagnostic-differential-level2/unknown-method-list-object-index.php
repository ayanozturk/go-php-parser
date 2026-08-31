<?php
class ListObjectService {}

/** @param list{ListObjectService} $objects */
function run(array $objects): void
{
    $objects[0]->missing();
}
