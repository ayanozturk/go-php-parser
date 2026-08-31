<?php
class CloneService {}

function run(CloneService $value): void
{
    (clone $value)->missing();
}
