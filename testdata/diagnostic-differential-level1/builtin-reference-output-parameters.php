<?php

function readBuiltinOutputs(): void
{
    preg_match_all('/item/', 'item item', $matches);
    echo $matches[0][0];

    preg_replace('/item/', 'entry', 'item', -1, count: $replacementCount);
    echo $replacementCount;

    sscanf('12 34', '%d %d', $first, $second);
    echo $first;
    echo $second;

    exec('printf ready', output: $commandOutput, result_code: $commandStatus);
    echo $commandOutput[0];
    echo $commandStatus;

    headers_sent(filename: $headerFile, line: $headerLine);
    echo $headerFile;
    echo $headerLine;
}
