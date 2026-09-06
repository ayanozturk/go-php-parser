<?php

enum CoverageRequestStatus: string
{
    case OPEN = 'open';

    public function canTransitionTo(self $newStatus): bool
    {
        return true;
    }
}

function transition(CoverageRequestStatus $from): bool
{
    return $from->canTransitionTo('claimed');
}
