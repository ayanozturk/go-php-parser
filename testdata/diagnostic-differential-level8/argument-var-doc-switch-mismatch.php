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
        switch ($action) {
            case 'go':
                $this->takesUser($this->getUser());
                break;
        }
    }
}
