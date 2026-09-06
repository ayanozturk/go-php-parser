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
    protected function createForm(string $type): FormInterface
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
