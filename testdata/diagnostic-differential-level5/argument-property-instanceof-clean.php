<?php

class CompanyTeamHealthOverview
{
}

class View
{
    public ?CompanyTeamHealthOverview $companyHealthOverview = null;
}

function createCompanyRows(CompanyTeamHealthOverview $overview): void
{
}

function run(View $view): void
{
    if ($view->companyHealthOverview instanceof CompanyTeamHealthOverview) {
        createCompanyRows($view->companyHealthOverview);
    }
}
