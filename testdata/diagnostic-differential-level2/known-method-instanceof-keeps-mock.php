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

class TemplateShift
{
    public function getName(): string
    {
        return '';
    }
}

final class ExampleTest extends \PHPUnit\Framework\TestCase
{
    public function testIt(?string $shiftName): void
    {
        $templateShift = $shiftName !== null ? $this->createMock(TemplateShift::class) : null;
        if ($templateShift instanceof TemplateShift) {
            $templateShift->method('getName');
        }
    }
}

}
