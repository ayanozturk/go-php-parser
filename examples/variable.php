<?php

$grade = 85.5;
$age = 25;
$greeting = "Hello World";
$isStudent = true;

$grades = [85, 92, 78, 95]; 
$person = new stdClass();
$person->name = $name;
$person->age = $age;
$person = null;

$grades = [85, 92, 78, 95];
$person = [
    $name,
    $age,
    $isStudent,
    $grade,
    $grades,
];

$a = $person[0];
$b = $person[1];
$c = $person[2];