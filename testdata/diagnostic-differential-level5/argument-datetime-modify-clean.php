<?php

function takesMutableDate(DateTime $value): void
{
}

function takesImmutableDate(DateTimeImmutable $value): void
{
}

function modifyDates(string $modifier): void
{
    takesMutableDate((new DateTime())->modify($modifier));
    takesImmutableDate((new DateTimeImmutable())->modify($modifier));
}
