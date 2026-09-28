<?php
class Relay { public function send(): string { return 'sent'; } }
class TestCase
{
    public function assertSame(mixed $expected, mixed $actual): void {}

    /** @phpstan-assert !null $actual */
    public function assertNotNull(mixed $actual): void {}
}
class Sender extends TestCase
{
    public ?Relay $relay;
    public function send(): string {
        $this->assertSame(new Relay(), $this->relay);
        $this->assertNotNull($this->relay);
        return $this->relay->send();
    }
}
