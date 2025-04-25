<?php

declare(strict_types=1);

namespace MyNamespace;

class MyClass
{
    private string $name;
    private int $age;

    public function __construct(string $name, int $age)
    {
        $this->name = $name;
        $this->age = $age;
    }

    public function getName(): string
    {
        return $this->name;
    }

    public function getAge(): int
    {
        return $this->age;
    }

    public function isAdult(): bool
    {
        return $this->age >= 18;
    }

    public function __toString(): string
    {
        return sprintf('MyClass(name: %s, age: %d)', $this->name, $this->age);
    }

    public function __invoke(string $greeting): string
    {
        return sprintf('%s %s', $greeting, $this->name);
    }

    public function __debugInfo(): array
    {
        return [
            'name' => $this->name,
            'age' => $this->age,
        ];
    }

    public function __serialize(): array
    {
        return [
            'name' => $this->name,
            'age' => $this->age,
        ];
    }

    public function __unserialize(array $data): void
    {
        $this->name = $data['name'];
        $this->age = $data['age'];
    }
}
