<?php

namespace Symfony\Component\HttpFoundation;

/**
 * @template TInput of string|int|float|bool|null
 */
final class InputBag
{
    /**
     * @template TDefault of string|int|float|bool|null
     *
     * @param TDefault $default
     *
     * @return TDefault|TInput
     */
    public function get(string $key, mixed $default = null): string|int|float|bool|null
    {
        return $default;
    }
}

class Request
{
    /** @var InputBag<string> */
    public InputBag $query;
}

function takesString(string $value): void
{
}

function search(Request $request): void
{
    takesString($request->query->get('search', ''));
}
