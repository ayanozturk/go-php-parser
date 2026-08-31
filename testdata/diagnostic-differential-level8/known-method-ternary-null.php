<?php
class TernaryService
{
    public function execute(): void {}
}

function run(bool $condition): void
{
    ($condition ? new TernaryService() : null)->execute();
}
