<?php

class ExampleClass {
    // Valid constants
    public const FOO = 'value';
    public const BAR_BAZ = 123;
    public const MY_CONSTANT = true;

    // Invalid constants
    public const invalidName = 'should fail';  // camelCase
    public const another_invalid = 'fail';     // camelCase with underscore
    public const INVALIDNAME = 'fail';         // no underscores
}

trait ExampleTrait {
    public const TRAIT_CONST = 'valid';
    public const invalid_trait_const = 'invalid'; // camelCase
}
