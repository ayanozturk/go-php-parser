<?php

class PreferenceFitScore
{
    public bool $hasPreferences = false;
}

function createPreferenceFitBadge(PreferenceFitScore $fit): void
{
}

function run(?PreferenceFitScore $preferenceFit): void
{
    createPreferenceFitBadge($preferenceFit);
}
