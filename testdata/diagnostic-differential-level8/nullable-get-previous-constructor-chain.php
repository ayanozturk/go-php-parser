<?php
$root = new Exception('root');
$mid = new Exception('mid', 0, $root);
$pay = new Exception('pay', 0, $mid);
$pay->getPrevious()->getPrevious()->getMessage();
