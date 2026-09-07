<?php
class Payload { public function toArray(): mixed { return []; } }
class Gateway
{
    /** @var Payload|null */
    public $payload;
    public function export(): mixed
    {
        if (method_exists($this->payload, 'toArray')) {
            return $this->payload->toArray();
        }
        return [];
    }
}
