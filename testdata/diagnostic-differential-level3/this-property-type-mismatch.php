<?php

final class TextRecord
{
    public string $text;

    public function replaceText(): void
    {
        $this->text = 42;
    }
}
