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
        $action = 'go';
        /** @var User $user */
        $user = $this->getUser();
        switch ($action) {
            case 'go':
                $this->takesUser($user);
                break;
        }
    }
}
