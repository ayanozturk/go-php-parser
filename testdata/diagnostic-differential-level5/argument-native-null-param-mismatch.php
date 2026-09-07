<?php

namespace Library;

class Token
{
    public function verify(string $otp, null|int $timestamp = null): bool
    {
        return true;
    }
}

function run(Token $token, string $otp): void
{
    $token->verify($otp, 'nope');
}
