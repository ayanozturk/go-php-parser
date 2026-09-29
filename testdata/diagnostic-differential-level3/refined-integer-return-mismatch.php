<?php

/** @return positive-int */
function zeroIsNotPositive(): int { return 0; }
/** @return int<1, 10> */
function outsideRange(): int { return 11; }
