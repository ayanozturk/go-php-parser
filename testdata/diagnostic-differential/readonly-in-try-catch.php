<?php
class ReadonlyAssignedAcrossTryCatch
{
    public readonly int $value;

    public function __construct(bool $fail)
    {
        try {
            if ($fail) {
                throw new RuntimeException();
            }
            $this->value = 1;
        } catch (RuntimeException $error) {
            $this->value = 2;
        }
    }
}
