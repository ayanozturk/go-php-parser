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
    save($passwordReset);
}
