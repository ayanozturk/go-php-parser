<?php

class Record {}

/**
 * @template T
 */
class EntityRepository
{
    /** @return T|null */
    public function find($id): ?object
    {
        return null;
    }
}

class ServiceEntityRepository extends EntityRepository
{
}

/**
 * @extends ServiceEntityRepository<Record>
 */
class RecordRepository extends ServiceEntityRepository
{
}

class Lookup
{
    public RecordRepository $repository;

    public function byId($id): ?Record
    {
        return $this->repository->find($id);
    }
}
