<?php
abstract class AbstractBase {
    abstract public function fromParent(): void;
}
class MissingParentMethod extends AbstractBase {}

