<?php
class Provider {}

function run(Provider $provider, string $method): void
{
    $provider->$method();
}
