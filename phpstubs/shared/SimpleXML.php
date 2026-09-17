<?php

class SimpleXMLElement implements Stringable, Countable, RecursiveIterator
{
    public function __construct(string $data, int $options = 0, bool $dataIsURL = false, string $namespaceOrPrefix = "", bool $isPrefix = false) {}

    public function addAttribute(string $qualifiedName, ?string $value = null, ?string $namespace = null): void {}

    public function addChild(string $qualifiedName, ?string $value = null, ?string $namespace = null): ?SimpleXMLElement { return null; }

    public function asXML(?string $filename = null): string|bool { return ""; }

    public function attributes(?string $namespaceOrPrefix = null, bool $isPrefix = false): ?SimpleXMLElement { return null; }

    public function children(?string $namespaceOrPrefix = null, bool $isPrefix = false): ?SimpleXMLElement { return null; }

    public function count(): int { return 0; }

    public function current(): mixed { return null; }

    public function getDocNamespaces(bool $recursive = false, bool $fromRoot = true): array|false { return []; }

    public function getName(): string { return ""; }

    public function getNamespaces(bool $recursive = false): array { return []; }

    public function getChildren(): ?SimpleXMLElement { return null; }

    public function hasChildren(): bool { return false; }

    public function key(): string|int { return 0; }

    public function next(): void {}

    public function registerXPathNamespace(string $prefix, string $namespace): bool { return true; }

    public function rewind(): void {}

    public function saveXML(?string $filename = null): string|bool { return ""; }

    public function __toString(): string { return ""; }

    public function valid(): bool { return true; }

    public function xpath(string $expression): array|null|false { return []; }
}

class SimpleXMLIterator extends SimpleXMLElement implements RecursiveIterator, Countable
{
}
