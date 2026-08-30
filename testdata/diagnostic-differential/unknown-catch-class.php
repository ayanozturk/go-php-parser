<?php

try {
    throw new Exception('fail');
} catch (MissingException $e) {
}
