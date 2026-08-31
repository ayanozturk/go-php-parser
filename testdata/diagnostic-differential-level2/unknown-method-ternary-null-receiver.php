<?php
class TernaryService {}

function run(bool $condition): void
{
    ($condition ? new TernaryService() : null)->missing();
}
