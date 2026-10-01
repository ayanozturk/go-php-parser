<?php

interface LeftContract {}
interface RightContract {}
final class BothContracts implements LeftContract, RightContract {}
final class OnlyLeftContract implements LeftContract {}

/** @template-covariant T */
interface NestedBox {}

/** @template T of NestedBox<LeftContract&RightContract> */
final class IntersectionBound {}
/** @template T of NestedBox<LeftContract|RightContract> */
final class UnionBound {}

/** @param IntersectionBound<NestedBox<BothContracts>> $value */
function acceptsNestedIntersection($value): void {}
/** @param UnionBound<NestedBox<OnlyLeftContract>> $value */
function acceptsNestedUnion($value): void {}
