<?php

class ResetPassword
{
}

function save(ResetPassword $reset): void
{
}

function findReset(): ?ResetPassword
{
    return null;
}

function run(): void
{
    $passwordReset = findReset();
    if (!$passwordReset) {
        $passwordReset = new ResetPassword();
    }
    save($passwordReset);
}
