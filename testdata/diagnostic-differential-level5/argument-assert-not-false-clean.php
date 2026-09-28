<?php

namespace PHPUnit\Framework {
    class Assert
    {
        /** @phpstan-assert !false $actual */
        public static function assertNotFalse(mixed $actual): void
        {
        }
    }
}

namespace {
    function requireDateTime(DateTime $date): void
    {
    }

    function run(DateTime|false $date): void
    {
        PHPUnit\Framework\Assert::assertNotFalse($date);
        requireDateTime($date);
    }
}
