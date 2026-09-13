<?php
namespace App;

use \Closure;

class Item {}

/** @param Closure(Item): (Item|null) $factory */
function acceptsClosure(Closure $factory): void {}

class Holder {
    /** @var \Closure(Item): Item */
    public Closure $factory;
}

/** @return Closure(Item): Item */
function returnsClosure(): Closure { throw new \RuntimeException(); }
