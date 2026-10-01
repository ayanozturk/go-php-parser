<?php
class InvalidCallableTemplate
{
    /**
     * @template-covariant T
     * @param T $value
     */
    public function provide($value): void {}
}
