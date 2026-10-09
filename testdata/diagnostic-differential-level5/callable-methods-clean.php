<?php
namespace Methods;
class Factory {
    /**
     * @template Outcome
     * @param (callable(): Outcome) $callback
     * @return Outcome
     */
    public function evaluate(callable $callback) { return $callback(); }
    /**
     * @template Payload
     * @param Payload $value
     * @return Payload
     */
    public function identity($value) { return $value; }
}
/** @template Item */
class Consumer {
    /** @param callable(Item): Item $callback */
    public function consume(callable $callback): void {}
}
/** @extends Consumer<int> */
class IntConsumer extends Consumer {}
function acceptInt(int $value): void {}
function acceptClosure(\Closure $value): void {}
function run(Factory $factory, IntConsumer $consumer, int $number): void {
    acceptInt($factory->evaluate(fn() => $number));
    acceptClosure($factory->identity(fn(): string => 'ready'));
    $consumer->consume(fn(int $value): int => $value);
}
class Item { public function id(): int { return 1; } }
class Initializer {
    /**
     * @template T of object
     * @param class-string<T> $class
     * @param \Closure(T): void $initialize
     * @return T
     */
    public function create(string $class, \Closure $initialize): object { return new $class; }
    /** @param \Closure(self): self $transform */
    public function accept(\Closure $transform): void {}
    public function local(): void { $this->accept(fn(self $value): self => $value); }
}
function initialized(Initializer $factory): int {
    return $factory->create(Item::class, function ($item) {})->id();
}
