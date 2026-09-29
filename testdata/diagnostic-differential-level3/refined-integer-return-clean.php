<?php

/** @return positive-int */
function positiveValue(): int { return 5; }
/** @return int<1, 10> */
function boundedValue(): int { return 10; }
/** @return non-negative-int */
function zeroValue(): int { return 0; }
