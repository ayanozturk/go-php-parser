<?php
class ReadonlyAssignedInFinally
{
    public readonly int $value;

    public function __construct()
    {
        try {
            throw new RuntimeException();
        } finally {
            $this->value = 1;
        }
    }
}
