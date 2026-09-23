<?php

namespace Illuminate\Support {
    /**
     * @template TKey of array-key
     * @template TValue
     * @implements \IteratorAggregate<TKey, TValue>
     */
    final class Collection implements \IteratorAggregate
    {
        /** @return \ArrayIterator<TKey, TValue> */
        public function getIterator(): \Traversable
        {
            return new \ArrayIterator([]);
        }
    }
}

namespace App {
    final class User
    {
        public function name(): string
        {
            return '';
        }
    }

    /** @param \Illuminate\Support\Collection<int, User> $users */
    function printUserNames(\Illuminate\Support\Collection $users): void
    {
        foreach ($users as $user) {
            $user->name();
        }
    }
}
