<?php
class LeftBranch {}
class RightBranch {}

function run(bool $condition): void
{
    ($condition ? new LeftBranch() : new RightBranch())->missing();
}
