<?php
class Beacon { public function ping(): string { return 'ready'; } }
function useBeacon(Beacon $beacon): string { return $beacon->ping(); }
class Harbor
{
    public ?Beacon $beacon;
    public Beacon $selected;
    public function inspect(): string {
        if ($this->beacon === null) { return ''; } else { return useBeacon($this->beacon); }
    }
}
