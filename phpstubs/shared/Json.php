<?php

interface JsonSerializable
{
    public function jsonSerialize(): mixed;
}

class JsonException extends Exception
{
}
