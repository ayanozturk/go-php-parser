<?php
class Payload { public function toArray(): mixed { return []; } }
class Gateway
{
    /** @var mixed */
    public $payload;
    public function export(): mixed
    {
        if (is_object($this->payload) && method_exists($this->payload, 'toArray')) {
            return $this->payload->toArray();
        }
        return [];
    }
}
