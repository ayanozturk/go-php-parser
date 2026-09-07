<?php

function makeDue(?DateTimeImmutable $due): void
{
}

function run(DateTimeImmutable|null|false $dueDate): void
{
    makeDue($dueDate);
}
