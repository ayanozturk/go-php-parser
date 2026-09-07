<?php

function makeDue(?DateTimeImmutable $due): void
{
}

function run(DateTimeImmutable|null|false $dueDate): void
{
    $actual = $dueDate === false ? new DateTimeImmutable() : $dueDate;
    makeDue($actual);
}
