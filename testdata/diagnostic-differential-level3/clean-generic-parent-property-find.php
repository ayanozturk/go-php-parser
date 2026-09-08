<?php

class User {}

/** @template T of object */
class Repo {
    /** @return T|null */
    public function find($id): ?object { return null; }
}

/** @extends Repo<User> */
class UserRepo extends Repo {}

class Service {
    public UserRepo $repository;

    public function byId($id): ?User {
        return $this->repository->find($id);
    }
}
