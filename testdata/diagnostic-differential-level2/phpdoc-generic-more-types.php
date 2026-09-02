<?php

/** @template TValue */
final class SingleTemplateBox
{
}

/** @param SingleTemplateBox<int, string> $box */
function inspectTooManyTemplateArguments(SingleTemplateBox $box): void
{
    echo 'checked';
}
