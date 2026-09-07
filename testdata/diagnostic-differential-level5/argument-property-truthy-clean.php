<?php

class Request
{
    public ?string $targetDateIso = null;
}

function run(Request $request): void
{
    if ($request->targetDateIso) {
        echo (new DateTimeImmutable($request->targetDateIso))->format('c');
    }
}
