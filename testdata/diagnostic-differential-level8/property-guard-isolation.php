<?php
class Beacon { public function ping(): string { return 'ready'; } }
function useBeacon(Beacon $beacon): string { return $beacon->ping(); }
class Harbor
{
    public ?Beacon $beacon;
    public Beacon $selected;
    public function inspect(): string {
        if ($this->beacon !== null) { $text = useBeacon($this->beacon); }
        return useBeacon($this->beacon);
    }
}
