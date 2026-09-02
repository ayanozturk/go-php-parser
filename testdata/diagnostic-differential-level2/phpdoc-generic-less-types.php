<?php

/**
 * @template TKey
 * @template TValue
 */
final class PairTemplateBox
{
}

/** @param PairTemplateBox<int> $box */
function inspectTooFewTemplateArguments(PairTemplateBox $box): void
{
    echo 'checked';
}
