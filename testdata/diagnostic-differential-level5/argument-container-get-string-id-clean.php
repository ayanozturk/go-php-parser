<?php

class UserRepository
{
}

interface ContainerInterface
{
    /**
     * @param string $id
     * @return object|null
     */
    public function get(string $id): ?object;
}

function run(ContainerInterface $container): void
{
    $repo = $container->get('app.user_repository');
    if ($repo === null) {
        return;
    }
}
