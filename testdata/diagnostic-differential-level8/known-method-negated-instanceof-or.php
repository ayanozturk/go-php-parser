<?php
class User
{
    public function getId(): string
    {
        return '';
    }
}

function run(?User $employee, User $user): void
{
    if (!$employee instanceof User || $employee->getId() !== $user->getId()) {
        return;
    }
}
