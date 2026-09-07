<?php
class Payload { public function toArray(): mixed { return []; } }
class Gateway
{
    /** @var Payload|null */
    public $payload;
    public function export(): mixed
    {
        return is_object($this->payload) && method_exists($this->payload, 'toArray')
            ? $this->payload->toArray()
            : [];
    }
}
