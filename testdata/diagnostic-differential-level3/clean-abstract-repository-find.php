<?php

interface EntityInterface {}

class Record implements EntityInterface {}

/**
 * @template T of object
 */
class EntityRepository
{
    /**
     * @return object|null
     * @phpstan-return T|null
     */
    public function find($id): ?object
    {
        return null;
    }
}

class ServiceEntityRepository extends EntityRepository
{
}

/**
 * @template T of EntityInterface
 * @template-extends ServiceEntityRepository<T>
 */
abstract class AbstractRepository extends ServiceEntityRepository
{
    /** @var EntityRepository<T>|null */
    private ?EntityRepository $resolvedRepository = null;

    /** @return T|null */
    public function find($id): ?object
    {
        return $this->resolveRepository()->find($id);
    }

    /** @return EntityRepository<T> */
    private function resolveRepository(): EntityRepository
    {
        if ($this->resolvedRepository instanceof EntityRepository) {
            return $this->resolvedRepository;
        }
        $this->resolvedRepository = new EntityRepository();
        return $this->resolvedRepository;
    }
}

/**
 * @extends AbstractRepository<Record>
 */
class RecordRepository extends AbstractRepository
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
