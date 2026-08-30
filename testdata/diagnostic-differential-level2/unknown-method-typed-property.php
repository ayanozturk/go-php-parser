<?php
class Service {}

class Holder
{
    public Service $service;
}

function run(Holder $holder): void
{
    $holder->service->missing();
}
