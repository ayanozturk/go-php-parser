<?php

final class PlainTemplateBox
{
}

/** @param PlainTemplateBox<int> $box */
function inspectNonGenericTemplate(PlainTemplateBox $box): void
{
    echo 'checked';
}
