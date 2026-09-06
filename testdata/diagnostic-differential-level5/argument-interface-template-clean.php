<?php

namespace Symfony\Contracts\EventDispatcher;

interface EventDispatcherInterface
{
    /**
     * @template T of object
     * @param T $event
     * @return T
     */
    public function dispatch(object $event, ?string $eventName = null): object;
}

namespace App;

use Symfony\Contracts\EventDispatcher\EventDispatcherInterface;

class PolicyCreatedEvent
{
}

function emit(EventDispatcherInterface $dispatcher): void
{
    $dispatcher->dispatch(new PolicyCreatedEvent());
}
