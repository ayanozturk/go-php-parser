<?php

class User
{
}

function requireUser(User $user): void
{
}

function run(?User $user): void
{
    requireUser($user);
}
