<?php

class InstanceWriter
{
    public function fill(&$out): void
    {
        $out = 'ready';
    }

    public function run(): void
    {
        $this->fill($value);
        echo $value;
    }
}
