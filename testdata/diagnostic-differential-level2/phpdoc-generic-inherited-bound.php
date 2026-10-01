<?php

class Animal {}
class Dog extends Animal {}

/** @template-covariant T */
interface ReadOnlyBox {}

/** @template-contravariant T */
interface WriteOnlyBox {}

/** @template T */
interface InvariantBox {}

/** @template-covariant T */
interface ParentBox {}

/** @implements ParentBox<Dog> */
final class DogParentBox implements ParentBox {}

/** @template T @implements ParentBox<T> */
class GenericChildBox implements ParentBox {}

/** @template T @extends GenericChildBox<T> */
final class GrandchildBox extends GenericChildBox implements ParentBox {}

/** @template T of ReadOnlyBox<Animal> */
final class CovariantBound {}

/** @template T of WriteOnlyBox<Dog> */
final class ContravariantBound {}

/** @template T of ParentBox<Animal> */
final class InheritedBound {}

/** @template T of InvariantBox<Animal> */
final class InvariantBound {}

/** @param CovariantBound<ReadOnlyBox<Dog>> $value */
function acceptsCovariantArgument($value): void {}

/** @param ContravariantBound<WriteOnlyBox<Animal>> $value */
function acceptsContravariantArgument($value): void {}

/** @param InheritedBound<DogParentBox> $value */
function acceptsFixedInheritedArgument($value): void {}

/** @param InheritedBound<GenericChildBox<Dog>> $value */
function acceptsSubstitutedInheritedArgument($value): void {}

/** @param InheritedBound<GrandchildBox<Dog>> $value */
function acceptsTransitivelySubstitutedInheritedArgument($value): void {}

/** @param InvariantBound<InvariantBox<Dog>> $value */
function rejectsInvariantArgument($value): void {}

/** @param CovariantBound<mixed> $value */
function rejectsMixedArgument($value): void {}
