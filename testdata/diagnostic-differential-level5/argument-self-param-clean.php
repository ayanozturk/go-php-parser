<?php

enum CoverageRequestStatus: string
{
    case OPEN = 'open';
    case CLAIMED = 'claimed';

    public function canTransitionTo(self $newStatus): bool
    {
        return true;
    }
}

final class StatusList
{
    public function merge(self $other): self
    {
        return $other;
    }
}

function transition(CoverageRequestStatus $from, CoverageRequestStatus $to): bool
{
    return $from->canTransitionTo($to);
}

function mergeLists(StatusList $left, StatusList $right): StatusList
{
    return $left->merge($right);
}
