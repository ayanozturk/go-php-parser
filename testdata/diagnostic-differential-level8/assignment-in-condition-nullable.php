<?php
$root = new Exception('root');
$e = new Exception('mid', 0, $root);
function readPrevious(Exception $e): string {
    if ($prev = $e->getPrevious()) {
        return $prev->getMessage();
    }
    return '';
}
