<?php

namespace App\Tests\Unit\EventSubscriber;

final class RememberMeResponseSubscriber
{
    public static function getSubscribedEvents(): array
    {
        return [];
    }
}

final class RememberMeResponseSubscriberTest
{
    public function testGetSubscribedEvents(): void
    {
        $subscriber = new RememberMeResponseSubscriber();
        $events = $subscriber::getSubscribedEvents();
        $this::getSubscribedEvents();
    }

    public static function getSubscribedEvents(): array
    {
        return [];
    }
}
