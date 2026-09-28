<?php

namespace Symfony\Component\DependencyInjection {
    interface ContainerInterface
    {
        /**
         * @template T of object
         * @param string $id
         * @phpstan-param class-string<T> $id
         * @return object|null
         * @phpstan-return T|null
         */
        public function get(string $id): ?object;
    }
}

namespace App {
    use Symfony\Component\DependencyInjection\ContainerInterface;

    final class Service {}

    function consume(Service $service): void {}

    function run(ContainerInterface $container): void
    {
        $service = $container->get(\App\Service::class);
        if ($service === null) {
            return;
        }
        consume($service);
    }
}
