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

function run(ContainerInterface $container): void
{
    $repo = $container->get('app.user_repository');
    if (!$repo) {
        return;
    }
}
