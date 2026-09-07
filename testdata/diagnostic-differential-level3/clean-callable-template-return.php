<?php

class Widget {}

interface Cache {
    /**
     * @template T
     * @param callable(): T $callback
     * @return T
     */
    public function get(string $key, callable $callback): mixed;
}

class Service {
    private Cache $cache;

    public function widget(): Widget {
        return $this->cache->get('k', function (): Widget {
            return new Widget();
        });
    }
}
