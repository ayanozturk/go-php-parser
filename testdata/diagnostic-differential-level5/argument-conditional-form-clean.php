<?php

namespace Symfony\Component\Form;

interface FormInterface
{
}

namespace Symfony\Bundle\FrameworkBundle\Controller;

use Symfony\Component\Form\FormInterface;

class AbstractController
{
    /**
     * @return ($type is class-string<FormFlowTypeInterface> ? FormFlowInterface : FormInterface)
     */
    protected function createForm(string $type, mixed $data = null, array $options = []): FormInterface
    {
        throw new \RuntimeException('stub');
    }
}

namespace App;

use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\Form\FormInterface;

class SearchType
{
}

class Demo extends AbstractController
{
    public function takesForm(FormInterface $form): void
    {
    }

    public function run(): void
    {
        $this->takesForm($this->createForm(SearchType::class));
    }
}
