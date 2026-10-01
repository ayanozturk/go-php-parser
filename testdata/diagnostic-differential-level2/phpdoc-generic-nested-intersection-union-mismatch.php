<?php

interface LeftContract {}
interface RightContract {}
final class OnlyLeftContract implements LeftContract {}
final class NeitherContract {}

/** @template-covariant T */
interface NestedBox {}

/** @template T of NestedBox<LeftContract&RightContract> */
final class IntersectionBound {}
/** @template T of NestedBox<LeftContract|RightContract> */
final class UnionBound {}

/** @param IntersectionBound<NestedBox<OnlyLeftContract>> $value */
function rejectsNestedIntersection($value): void {}
/** @param UnionBound<NestedBox<NeitherContract>> $value */
function rejectsNestedUnion($value): void {}
