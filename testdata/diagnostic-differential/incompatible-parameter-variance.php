<?php

class AnimalReceiver
{
}

class DogReceiver extends AnimalReceiver
{
}

interface AcceptsAnyAnimal
{
    public function accept(AnimalReceiver $value): void;
}

final class AcceptsOnlyDog implements AcceptsAnyAnimal
{
    public function accept(DogReceiver $value): void
    {
    }
}
