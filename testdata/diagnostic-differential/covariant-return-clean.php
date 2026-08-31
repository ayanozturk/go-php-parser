<?php
interface Handler {
    public function handle(): mixed;
}
class SpecificHandler implements Handler {
    public function handle(): string { return "ok"; }
}

