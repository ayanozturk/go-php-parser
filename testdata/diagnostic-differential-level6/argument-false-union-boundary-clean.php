<?php

function requireNullableDateTime(?DateTimeImmutable $date): void
{
}

function run(DateTimeImmutable|null|false $date): void
{
    requireNullableDateTime($date);
}
