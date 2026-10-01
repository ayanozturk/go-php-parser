<?php
/** @template-covariant T */
interface Producer
{
    /** @return T|null */
    public function get();
}

/** @template-contravariant T */
interface Consumer
{
    /** @param T $value */
    public function accept($value): void;
}
