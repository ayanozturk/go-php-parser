<?php

class ReflectionException extends Exception
{
}

class ReflectionType
{
    public function allowsNull(): bool
    {
        return false;
    }

    public function __toString(): string
    {
        return '';
    }
}

class ReflectionNamedType extends ReflectionType
{
    public function getName(): string
    {
        return '';
    }

    public function isBuiltin(): bool
    {
        return false;
    }
}

class ReflectionFunctionAbstract
{
    public function getName(): string
    {
        return '';
    }

    public function getNumberOfParameters(): int
    {
        return 0;
    }

    /** @return ReflectionParameter[] */
    public function getParameters(): array
    {
        return [];
    }

    public function getReturnType(): ?ReflectionType
    {
        return null;
    }

    public function hasReturnType(): bool
    {
        return false;
    }

    public function getDocComment(): string|false
    {
        return false;
    }

    public function getFileName(): string|false
    {
        return false;
    }

    public function getStartLine(): int|false
    {
        return false;
    }

    public function getEndLine(): int|false
    {
        return false;
    }

    public function isDeprecated(): bool
    {
        return false;
    }

    public function isVariadic(): bool
    {
        return false;
    }

    public function isGenerator(): bool
    {
        return false;
    }

    public function returnsReference(): bool
    {
        return false;
    }
}

class ReflectionParameter
{
    public function getName(): string
    {
        return '';
    }

    public function getType(): ?ReflectionType
    {
        return null;
    }

    public function hasType(): bool
    {
        return false;
    }

    public function isOptional(): bool
    {
        return false;
    }

    public function isPassedByReference(): bool
    {
        return false;
    }

    public function isDefaultValueAvailable(): bool
    {
        return false;
    }

    public function getDefaultValue(): mixed
    {
        return null;
    }

    public function allowsNull(): bool
    {
        return false;
    }

    public function isVariadic(): bool
    {
        return false;
    }

    public function isPromoted(): bool
    {
        return false;
    }
}

class ReflectionMethod extends ReflectionFunctionAbstract
{
    public function __construct($objectOrMethod, $method = null)
    {
    }

    public function isPublic(): bool
    {
        return false;
    }

    public function isPrivate(): bool
    {
        return false;
    }

    public function isProtected(): bool
    {
        return false;
    }

    public function isStatic(): bool
    {
        return false;
    }

    public function isAbstract(): bool
    {
        return false;
    }

    public function isFinal(): bool
    {
        return false;
    }

    public function isConstructor(): bool
    {
        return false;
    }

    public function isDestructor(): bool
    {
        return false;
    }

    public function getDeclaringClass(): ReflectionClass
    {
        return new ReflectionClass('stdClass');
    }

    public function invoke($object, ...$args): mixed
    {
        return null;
    }

    public function invokeArgs($object, array $args): mixed
    {
        return null;
    }

    public function setAccessible(bool $accessible): void
    {
    }
}

class ReflectionProperty
{
    public function __construct($class, $property)
    {
    }

    public function getName(): string
    {
        return '';
    }

    public function getValue($object = null): mixed
    {
        return null;
    }

    public function setValue($objectOrValue, $value = null): void
    {
    }

    public function isPublic(): bool
    {
        return false;
    }

    public function isPrivate(): bool
    {
        return false;
    }

    public function isProtected(): bool
    {
        return false;
    }

    public function isStatic(): bool
    {
        return false;
    }

    public function isReadOnly(): bool
    {
        return false;
    }

    public function getType(): ?ReflectionType
    {
        return null;
    }

    public function hasType(): bool
    {
        return false;
    }

    public function getDeclaringClass(): ReflectionClass
    {
        return new ReflectionClass('stdClass');
    }

    public function getAttributes($name = null, int $flags = 0): array
    {
        return [];
    }
}

class ReflectionClass
{
    public function __construct($objectOrClass)
    {
    }

    public function getName(): string
    {
        return '';
    }

    public function hasMethod(string $name): bool
    {
        return false;
    }

    public function getMethod(string $name): ReflectionMethod
    {
        return new ReflectionMethod($this->getName(), $name);
    }

    /** @return ReflectionMethod[] */
    public function getMethods(?int $filter = null): array
    {
        return [];
    }

    public function hasProperty(string $name): bool
    {
        return false;
    }

    public function getProperty(string $name): ReflectionProperty
    {
        return new ReflectionProperty($this->getName(), $name);
    }

    /** @return ReflectionProperty[] */
    public function getProperties(?int $filter = null): array
    {
        return [];
    }

    public function getConstructor(): ?ReflectionMethod
    {
        return null;
    }

    public function getConstant(string $name): mixed
    {
        return false;
    }

    public function hasConstant(string $name): bool
    {
        return false;
    }

    public function isFinal(): bool
    {
        return false;
    }

    public function isAbstract(): bool
    {
        return false;
    }

    public function isReadOnly(): bool
    {
        return false;
    }

    public function isSubclassOf($class): bool
    {
        return false;
    }

    public function newInstance(...$args): object
    {
        return new stdClass();
    }

    public function getAttributes($name = null, int $flags = 0): array
    {
        return [];
    }
}

class ReflectionObject extends ReflectionClass
{
    public function __construct($object)
    {
    }
}

class ReflectionFunction extends ReflectionFunctionAbstract
{
    public function __construct($function)
    {
    }

    public function invoke(...$args): mixed
    {
        return null;
    }
}
