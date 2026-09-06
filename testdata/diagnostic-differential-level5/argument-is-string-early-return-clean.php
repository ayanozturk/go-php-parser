<?php

function parseCalendarDate(string $value): void
{
}

function run(?string $start, ?string $end): void
{
    if (!is_string($start) || !is_string($end)) {
        return;
    }
    parseCalendarDate($start);
    parseCalendarDate($end);
}
