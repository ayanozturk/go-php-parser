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
        $user = $this->getUser();
        $this->takesUser($user);
    }
}
