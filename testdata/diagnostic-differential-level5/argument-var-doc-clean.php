<?php

class User
{
}

class Controller
{
    public function getUser(): ?User
    {
        return null;
    }

    public function takesUser(User $user): void
    {
    }

    public function run(): void
    {
        /** @var User $user */
        $user = $this->getUser();
        $this->takesUser($user);
    }
}
