<?php

/**
 * @phpstan-type ManagerSummaries list<string>
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
    return new CompanyOverview('not-an-array');
}
