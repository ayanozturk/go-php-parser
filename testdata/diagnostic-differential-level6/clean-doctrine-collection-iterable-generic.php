<?php

namespace Doctrine\Common\Collections;

/** @template TKey of array-key @template T @extends \IteratorAggregate<TKey, T> */
interface Collection extends \IteratorAggregate {}

namespace App;

use Doctrine\Common\Collections\Collection;

final class User {}

/** @param Collection<string, User> $items */
function consume(Collection $items): void
{
    echo count($items);
}
