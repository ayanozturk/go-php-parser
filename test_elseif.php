<?php

if ($condition1) {
    echo "condition1 is true";
} else if ($condition2) {
    echo "condition2 is true";
} elseif ($condition3) {
    echo "condition3 is true";
} else   if ($condition4) {
    echo "condition4 is true";
} else {
    echo "all conditions are false";
}

// This should be ignored
$message = "Use else if carefully in strings";

/* This else if should also be ignored */

if ($nested) {
    if ($inner) {
        echo "inner";
    } else if ($inner2) {
        echo "inner2";
    }
}