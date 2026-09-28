<?php
class Policy { public function getLimit(): int { return 0; } }
class Evaluator
{
    /** @var mixed */
    public $policy;
    public function limit(): int
    {
        if ($this->policy && is_object($this->policy) && method_exists($this->policy, 'getLimit')) {
            return $this->policy->getLimit();
        }
        return 0;
    }
}
