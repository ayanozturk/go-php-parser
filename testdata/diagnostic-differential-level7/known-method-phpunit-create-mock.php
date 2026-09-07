<?php

namespace PHPUnit\Framework\MockObject {

interface MockObject
{
    public function method(string $name): MockObject;
}

}

namespace PHPUnit\Framework {

use PHPUnit\Framework\MockObject\MockObject;

abstract class TestCase
{
    /**
     * @template RealInstanceType of object
     * @param class-string<RealInstanceType> $type
     * @return MockObject&RealInstanceType
     */
    final protected function createMock(string $type): MockObject
    {
        throw new \RuntimeException('stub');
    }
}

}

namespace {

class User
{
    public function id(): string
    {
        return '';
    }
}

final class UserTest extends \PHPUnit\Framework\TestCase
{
    public function testMock(): void
    {
        $user = $this->createMock(User::class);
        $user->method('id');
        $user->id();
    }
}

}
