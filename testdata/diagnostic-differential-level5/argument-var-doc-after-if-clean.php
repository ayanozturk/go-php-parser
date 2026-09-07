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

    public function run(array $items): void
    {
        if (empty($items)) {
            return;
        }
        /** @var User $user */
        $user = $this->getUser();
        switch ('go') {
            case 'go':
                $this->takesUser($user);
                break;
        }
    }
}
