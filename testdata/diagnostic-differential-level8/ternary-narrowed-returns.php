<?php
class Lantern
{
    public function glow(): string { return 'lit'; }
}
function illuminate(Lantern $lamp): string { return $lamp->glow(); }
function inspect(?Lantern $lamp): Lantern
{
    return $lamp === null ? new Lantern() : $lamp;
}
