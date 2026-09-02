<?php

final class KnownDocumentedType
{
}

/**
 * @param KnownDocumentedType $value
 * @return KnownDocumentedType
 */
function passKnownDocumentedValue(KnownDocumentedType $value): KnownDocumentedType
{
    echo 'checked';
    return $value;
}
