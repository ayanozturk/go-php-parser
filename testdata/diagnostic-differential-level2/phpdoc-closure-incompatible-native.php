<?php
class Item {}

/** @param Closure(Item): Item $factory */
function incompatibleClosure(int $factory): void {}
