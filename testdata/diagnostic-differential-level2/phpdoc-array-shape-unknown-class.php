<?php

/** @param array{service: MissingShapeService} $config */
function inspectShapeClass(array $config): void
{
    echo isset($config['service']);
}
