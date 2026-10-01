<?php

function acceptConditionalInt(int $value): void {}
function acceptConditionalString(string $value): void {}

/**
 * @param bool $asInt
 * @return ($asInt is true ? int : string)
 */
function conditionalValue($asInt)
{
    if ($asInt) {
        return 1;
    }
    return 'text';
}

class ConditionalProvider
{
    /**
     * @param bool $asInt
     * @return ($asInt is true ? int : string)
     */
    public function value($asInt)
    {
        return $asInt ? 1 : 'text';
    }
}

$provider = new ConditionalProvider();
acceptConditionalString(conditionalValue(true));
acceptConditionalInt(conditionalValue(false));
acceptConditionalString($provider->value(true));
acceptConditionalInt($provider->value(false));
