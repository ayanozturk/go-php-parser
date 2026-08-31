<?php
class CoalesceService {}

function run(?CoalesceService $value): void
{
    ($value ?? new CoalesceService())->missing();
}
