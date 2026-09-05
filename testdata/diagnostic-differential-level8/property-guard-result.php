<?php
class Beacon { public function ping(): string { return 'ready'; } }
function useBeacon(Beacon $beacon): string { return $beacon->ping(); }
class Harbor
{
    public ?Beacon $beacon;
    public Beacon $selected;
    public function inspect(): Beacon {
        return $this->beacon ?: new Beacon();
    }
    public function choose(): Beacon {
        if ($this->beacon !== null) { return $this->beacon; }
        return new Beacon();
    }
}
