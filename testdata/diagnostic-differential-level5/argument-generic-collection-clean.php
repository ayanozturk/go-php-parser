<?php

namespace Doctrine\Common\Collections;

/**
 * @template TKey of array-key
 * @template T
 */
interface Collection
{
}

/**
 * @template TKey of array-key
 * @template T
 * @implements Collection<TKey, T>
 */
class ArrayCollection implements Collection
{
}

class User
{
}

class Holder
{
    /** @param Collection<string, User> $users */
    public function setUsers($users): void
    {
    }
}

function run(Holder $holder): void
{
    $holder->setUsers(new ArrayCollection());
}
