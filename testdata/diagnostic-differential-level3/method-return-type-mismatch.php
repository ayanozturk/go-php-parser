<?php

final class ReturnTypeExample
{
    public function declaredIntegerButReturnsText(): int
    {
        return 'text';
    }
}
