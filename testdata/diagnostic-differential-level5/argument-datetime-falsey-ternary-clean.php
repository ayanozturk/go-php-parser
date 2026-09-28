<?php

function acceptsNullableDateTime(?DateTime $date): void
{
}

function run(): void
{
    acceptsNullableDateTime(DateTime::createFromFormat('!Y-m-d', '2026-01-02') ?: null);
}
