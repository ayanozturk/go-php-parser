<?php

class Factory
{
    /** @return self */
    public function selfValue() { return new self(); }
    /** @return static */
    public function staticValue() { return $this; }
}

class ChildFactory extends Factory {}

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

class ChildFluentFactory extends FluentFactory {}

function acceptFactory(Factory $value): void {}
function acceptChildFactory(ChildFactory $value): void {}
function acceptChildFluent(ChildFluentFactory $value): void {}

function valid(ChildFactory $child, ParentFactory $parent, ChildFluentFactory $fluent): void
{
    acceptFactory($child->selfValue());
    acceptChildFactory($child->staticValue());
    acceptFactory($parent->parentValue());
    acceptChildFluent($fluent->thisValue());
}
