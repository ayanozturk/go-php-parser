<?php
class ChainService {}

class ChainFactory
{
    public function service(): ChainService
    {
        return new ChainService();
    }
}

(new ChainFactory())->service()->missing();
