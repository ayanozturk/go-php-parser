<?php
class IntListLeft {}
class IntListRight {}

/**
 * @param list{callable(): IntListLeft, callable(): IntListRight} $list
 */
function run(array $list, int $i): void
{
    $list[$i]()->missing();
}
