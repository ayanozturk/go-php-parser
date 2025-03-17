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
    public function myMethod();
}

class MyClass implements MyInterface {
    public function sayHello(string $name) {
        echo "Hello, world, $name!";
    }
}

$myClass = new MyClass();
$myClass->sayHello("John Doe");