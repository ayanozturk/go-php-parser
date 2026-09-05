<?php
class Lantern
{
    public function glow(): string { return 'lit'; }
}
function illuminate(Lantern $lamp): string { return $lamp->glow(); }
function inspect(?Lantern $lamp): string
{
    $text = $lamp ? $lamp->glow() : '';
    return illuminate($lamp);
}
