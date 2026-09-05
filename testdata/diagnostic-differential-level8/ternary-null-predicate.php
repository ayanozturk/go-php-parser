<?php
class Lantern
{
    public function glow(): string { return 'lit'; }
}
function illuminate(Lantern $lamp): string { return $lamp->glow(); }
function inspect(?Lantern $lamp): string
{
    return is_null($lamp) ? '' : $lamp->glow();
}
