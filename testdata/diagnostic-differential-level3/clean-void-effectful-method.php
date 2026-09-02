<?php

final class StatefulVoidWorker
{
    public int $state = 0;

    public function finish(): void
    {
        $this->state = 1;
    }
}
