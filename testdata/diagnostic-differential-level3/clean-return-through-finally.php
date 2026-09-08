<?php

function load_label(bool $useFallback): string
{
    try {
        if ($useFallback) {
            throw new RuntimeException();
        }

        return 'primary';
    } catch (RuntimeException $exception) {
        return 'fallback';
    } finally {
        record_cleanup();
    }
}

function record_cleanup(): void
{
}
