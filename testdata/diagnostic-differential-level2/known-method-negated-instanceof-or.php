<?php
interface UserInterface {}

class User implements UserInterface
{
    public function isAdmin(): bool
    {
        return false;
    }
}

function run(?UserInterface $user): void
{
    if (!$user instanceof User || !$user->isAdmin()) {
        return;
    }
}
