<?php
$root = new Exception('root');
$mid = new Exception('mid', 0, $root);
$pay = new Exception('pay', 0, $mid);
$previous = $pay->getPrevious();
if ($previous !== null) {
    $previous->getPrevious()?->getMessage();
}
