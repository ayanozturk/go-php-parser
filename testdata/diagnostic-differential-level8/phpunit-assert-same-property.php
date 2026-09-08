<?php
class Relay { public function send(): string { return 'sent'; } }
class TestCase { public function assertSame(mixed $expected, mixed $actual): void {} }
class Sender extends TestCase
{
    public ?Relay $relay;
    public function send(): string {
        $this->assertSame(new Relay(), $this->relay);
        return $this->relay->send();
    }
}
