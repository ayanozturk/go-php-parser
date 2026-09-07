<?php

class Request
{
    public ?string $targetDateIso = null;
}

function run(Request $request): void
{
    echo (new DateTimeImmutable($request->targetDateIso))->format('c');
}
