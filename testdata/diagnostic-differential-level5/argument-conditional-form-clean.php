<?php

namespace Symfony\Component\Form;

interface FormInterface
{
}

interface FormFlowInterface extends FormInterface
{
}

interface FormFlowTypeInterface
{
}

namespace Symfony\Bundle\FrameworkBundle\Controller;

use Symfony\Component\Form\FormInterface;
use Symfony\Component\Form\FormFlowInterface;
use Symfony\Component\Form\FormFlowTypeInterface;

class AbstractController
{
    /**
     * @return ($type is class-string<FormFlowTypeInterface> ? FormFlowInterface : FormInterface)
     */
    protected function createForm(string $type, mixed $data = null, array $options = []): FormFlowInterface|FormInterface
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
