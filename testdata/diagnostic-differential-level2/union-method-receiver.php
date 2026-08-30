<?php
class FirstChoice {}

class SecondChoice
{
    public function optional(): void {}
}

function run(FirstChoice|SecondChoice $choice): void
{
    $choice->optional();
}
