<?php

namespace PHPUnit\Framework\MockObject;

interface MockObject
{
    public function expects(mixed $matcher): MockObject;

    public function method(string $name): MockObject;
}

namespace {

use PHPUnit\Framework\MockObject\MockObject;

class Service
{
    public function save(): void
    {
    }
}

class Example
{
    /** @var Service&MockObject */
    private Service $service;

    public function testIt(): void
    {
        $this->service->expects(null)->method('save');
    }
}

}
