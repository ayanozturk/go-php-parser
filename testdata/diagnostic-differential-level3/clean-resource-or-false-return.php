<?php

/** @return resource|false */
function open_read_stream(string $path)
{
    if ($path === '') {
        return false;
    }

    return fopen($path, 'rb');
}
