<?php

// Example PHP file to test function parsing

function sayHello($name) {
    echo "Hello, $name!";
}

function add($a, $b) {
    $sum = $a + $b;
    return $sum;
}

function greet($greeting = "Hello", $name = "World") {
    echo "$greeting, $name!";
}

function calculateArea(float $length, float $width): float {
    return $length * $width;
}

function getUserInfo(): array {
    return [
        'name' => 'John Doe',
        'age' => 30,
        'email' => 'john.doe@example.com'
    ];
}

// function variadicExample(...$args) {
    // foreach ($args as $arg) {
        // echo $arg . "\n";
    // }
// }

sayHello("Alice");
add(5, 10);
greet();
calculateArea(5.5, 3.2);
getUserInfo();
// variadicExample(1, 2, 3, 4, 5);
