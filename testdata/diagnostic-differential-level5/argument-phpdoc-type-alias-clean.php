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

function run(): void
{
    new CompanyOverview([new ManagerSummary()]);
}
