<?php
class ReadonlyConditionallyAssignedInFinally
{
    public readonly int $value;

    public function __construct(bool $initialize)
    {
        try {
            echo "work";
        } finally {
            if ($initialize) {
                $this->value = 1;
            }
        }
    }
}
