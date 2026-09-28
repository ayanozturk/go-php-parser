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
    protected function createForm(string $type): FormFlowInterface|FormInterface
    {
        throw new \RuntimeException('stub');
    }
}

namespace App;

use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;

class Demo extends AbstractController
{
    public function takesString(string $value): void
    {
    }

    public function run(): void
    {
        $this->takesString($this->createForm('SearchType'));
    }
}
