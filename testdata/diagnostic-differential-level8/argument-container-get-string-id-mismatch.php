<?php

class UserRepository
{
}

interface ContainerInterface
{
    /** @return object|null */
    public function get(string $id): ?object;
}

function notify(UserRepository $repo): void
{
}

function run(ContainerInterface $container): void
{
    notify($container->get('app.user_repository'));
}
