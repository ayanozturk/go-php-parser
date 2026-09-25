<?php

namespace PHPUnit\Framework\Attributes {
    #[\Attribute(\Attribute::TARGET_METHOD)]
    final class DataProvider
    {
        public function __construct(string $methodName)
        {
        }
    }
}

namespace Tests {
    use PHPUnit\Framework\Attributes\DataProvider;

    final class ProviderTest
    {
        #[DataProvider('stringCases')]
        /** @param list<string> $values */
        public function testValues(array $values): void
        {
        }

        /** @return iterable<array{list<string>}> */
        public static function stringCases(): iterable
        {
            yield [['one', 'two']];
        }
    }
}
