<?php
namespace Demo;

const KNOWN_LEVEL_ONE_VALUE = 1;

function useConstants(): void {
    echo KNOWN_LEVEL_ONE_VALUE;
    echo MISSING_LEVEL_ONE_VALUE;
    echo __DIR__, __FILE__, __LINE__, __NAMESPACE__, __FUNCTION__;
    echo match (true) {
        default => 'fallback',
    };
}

class MagicConstantContext {
    public function values(): void { echo __CLASS__, __METHOD__; }
}

trait MagicConstantTrait {
    public function traitName(): void { echo __TRAIT__; }
}
