<?php

class Subscription
{
}

interface ObjectRepository
{
    /**
     * @param string $className
     * @phpstan-param class-string<T> $className
     * @return object|null
     * @phpstan-return T|null
     * @template T of object
     */
    public function find(string $className, mixed $id): object|null;
}

function notify(Subscription $subscription): void
{
}

function run(ObjectRepository $em): void
{
    $subscription = $em->find(Subscription::class, 1);
    if (!$subscription) {
        return;
    }
    notify($subscription);
}
