<?php
// This is a comment
function MyFunction($name) {
    $var = "Hello";
    $var2 = 123;
    $var3 = $name;

    if (true) {
        echo "Hello, $name!";
    }
}

interface MyInterface {
    public function myMethod(string $name, int $age): string;
    public function getData(array $options = []): array;
}

class MyClass implements MyInterface {
    public function myMethod(string $name, int $age): string
    {
        $people = ["John", "Jane", "Jim", "Jill"];

        return "Hello, $name! You are $age years old.";
    }

    public function getData(array $options = []): array {
        return ['name' => 'John', 'age' => 30];
    }

    public function sayHello(string $name) {
        echo "Hello, world, $name!";
    }
}

$myClass = new MyClass();
$myClass->sayHello("John Doe");
$result = $myClass->myMethod("Jane", 25);
$data = $myClass->getData();