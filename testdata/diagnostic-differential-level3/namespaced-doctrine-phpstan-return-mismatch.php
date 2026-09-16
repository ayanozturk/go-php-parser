<?php

namespace Doctrine\ORM;

/**
 * @template T of object
 */
class EntityRepository
{
    /**
     * @return object|null
     * @phpstan-return ?T
     */
    public function find(mixed $id): object|null
    {
        return null;
    }
}

namespace Doctrine\Bundle\DoctrineBundle\Repository;

use Doctrine\ORM\EntityRepository;

/**
 * @template T of object
 * @template-extends EntityRepository<T>
 */
class ServiceEntityRepository extends EntityRepository
{
}

namespace App\Entity;

class DocumentPolicy
{
}

namespace App\Repository;

use App\Entity\DocumentPolicy;
use Doctrine\Bundle\DoctrineBundle\Repository\ServiceEntityRepository;

/**
 * @extends ServiceEntityRepository<DocumentPolicy>
 */
class DocumentPolicyRepository extends ServiceEntityRepository
{
}

namespace App\Service;

use App\Repository\DocumentPolicyRepository;

class DocumentPolicyService
{
    public function __construct(private readonly DocumentPolicyRepository $repository)
    {
    }

    public function getPolicyById(string $id): string
    {
        return $this->repository->find($id);
    }
}
