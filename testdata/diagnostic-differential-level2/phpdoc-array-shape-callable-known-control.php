<?php

final class KnownShapeService
{
}

/** @param array{factory: callable(): KnownShapeService} $config */
function inspectKnownShapeCallable(array $config): void
{
    $service = ($config['factory'])();
    echo $service instanceof KnownShapeService;
}
