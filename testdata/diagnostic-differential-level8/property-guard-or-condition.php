<?php
class Beacon { public function ping(): string { return 'ready'; } }
function useBeacon(Beacon $beacon): string { return $beacon->ping(); }
class Harbor
{
    public ?Beacon $beacon;
    public Beacon $selected;
    public function inspect(bool $flag): string {
        if ($this->beacon !== null || $flag) { return useBeacon($this->beacon); }
        return '';
    }
}
