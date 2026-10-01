<?php
/** @template-covariant T */
interface Producer
{
    /** @param T $value */
    public function replace($value): void;
}
