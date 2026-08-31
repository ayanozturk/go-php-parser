<?php
class MatchLeft {}
class MatchRight {}

function run(bool $flag): void
{
    (match ($flag) {
        true => new MatchLeft(),
        false => new MatchRight(),
    })->missing();
}
