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
    createCompanyRows($view->companyHealthOverview);
}
