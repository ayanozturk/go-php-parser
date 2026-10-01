<?php

class Factory
{
    /** @return self */
    public function selfValue() { return new self(); }
}

class ChildFactory extends Factory {}

class OtherFactory {}

class ParentFactory extends Factory
{
    /** @return parent */
    public function parentValue() { return new Factory(); }
}

class FluentFactory
{
    /** @return $this */
    public function thisValue() { return $this; }
}

function acceptOtherFactory(OtherFactory $value): void {}
function acceptOtherFluent(OtherFluentFactory $value): void {}

class ChildFluentFactory extends FluentFactory {}
class OtherFluentFactory {}

function invalid(ChildFactory $child, ParentFactory $parent, FluentFactory $fluent): void
{
    acceptOtherFactory($child->selfValue());
    acceptOtherFactory($parent->parentValue());
    acceptOtherFluent($fluent->thisValue());
}
