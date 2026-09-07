<?php
class Policy { public function getLimit(): int { return 0; } }
class Evaluator
{
    /** @var Policy|null */
    public $policy;
    public function limit(): int
    {
        if ($this->policy && method_exists($this->policy, 'getLimit')) {
            return $this->policy->getLimit();
        }
        return 0;
    }
}
