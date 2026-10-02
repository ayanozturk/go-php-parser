<?php
$value = 1;
$used = 2;
$callback = function () use ($value, $used): int { return $used; };
