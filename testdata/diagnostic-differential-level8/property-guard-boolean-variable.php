<?php
class Beacon { public function ping(): string { return 'ready'; } }
class Harbor
{
    public ?Beacon $beacon;
    public function inspect(): string {
        $ready = $this->beacon !== null && $this->beacon;
        if ($ready) {
            return $this->beacon->ping();
        }
        return '';
    }
}
