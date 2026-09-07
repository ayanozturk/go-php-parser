<?php

class User {}

/** @template T */
class Repository {
    /** @return T|null */
    public function find(mixed $id): ?object { return null; }
}

/** @extends Repository<User> */
class UserRepository extends Repository {
    public function byId(mixed $id): ?User {
        return $this->find($id);
    }
}
