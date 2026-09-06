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

function run(): void
{
    new CompanyOverview('not-an-array');
}
