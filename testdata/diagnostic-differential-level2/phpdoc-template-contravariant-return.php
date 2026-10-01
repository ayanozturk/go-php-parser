<?php
/** @template-contravariant T */
interface Consumer
{
    /** @return T */
    public function expose();
}
