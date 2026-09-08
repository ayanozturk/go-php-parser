<?php

namespace Differential\Level3\ClassTemplateReturn;

/** @template T of object */
abstract class Store
{
    /** @return T|null */
    abstract protected function stored(): ?object;

    /** @return T|null */
    public function current(): ?object
    {
        return $this->stored();
    }
}
