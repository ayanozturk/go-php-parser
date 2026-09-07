<?php
class Service {}

function run(Service $service): void
{
    if (method_exists($service, 'optional')) {
        $service->optional();
    }
}
