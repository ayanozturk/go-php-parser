<?php

function takesParsedDate(DateTime $value): void
{
}

/** @return DateTime|false */
function parseDate(string $value)
{
    return DateTime::createFromFormat('Y-m-d', $value);
}

function useParsedDate(string $value): void
{
    takesParsedDate(parseDate($value));
}
