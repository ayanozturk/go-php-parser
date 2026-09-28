<?php
class Beacon { public function ping(): string { return 'ready'; } }
class Harbor
{
    public bool $enabled = true;
    public ?Beacon $beacon;
    public function inspect(): string {
        $ready = $this->enabled && $this->beacon !== null;
        if ($ready) {
            return $this->beacon->ping();
        }
        return '';
    }
}
