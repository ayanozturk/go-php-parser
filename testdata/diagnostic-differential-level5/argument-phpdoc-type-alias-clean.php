<?php

class ManagerSummary
{
}

/**
 * @phpstan-type ManagerSummaries list<ManagerSummary>
 */
final readonly class CompanyOverview
{
    /**
     * @param ManagerSummaries $managers
     */
    public function __construct(public array $managers)
    {
    }
}

function run(): CompanyOverview
{
    return new CompanyOverview([new ManagerSummary()]);
}
