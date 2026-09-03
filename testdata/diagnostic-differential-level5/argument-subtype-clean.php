<?php

class Animal {}

final class Dog extends Animal {}

function acceptAnimal(Animal $animal): void
{
    echo get_class($animal);
}

function runSubtype(): void
{
    acceptAnimal(new Dog());
}
