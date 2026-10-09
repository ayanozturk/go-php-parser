<?php
namespace Methods;
class Factory {
    /**
     * @template Outcome
     * @param callable(): Outcome $callback
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
function wrong(Factory $factory, IntConsumer $consumer): void {
    acceptInt($factory->evaluate(fn() => 'wrong'));
    $consumer->consume(fn(string $value): string => $value);
}
