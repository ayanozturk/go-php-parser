<?php

class UserRepository
{
}

interface ContainerInterface
{
    /**
     * @param string $id
     * @phpstan-param class-string<T> $id
     * @return object|null
     * @phpstan-return T|null
     * @template T of object
     */
    public function get(string $id): ?object;
}

function notify(UserRepository $repo): void
{
}

function run(ContainerInterface $container): void
{
    notify($container->get('app.user_repository'));
}
