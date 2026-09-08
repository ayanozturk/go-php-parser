<?php

namespace Fixture\Catalog;

final class Entry
{
}

final class EntryPage
{
    /**
     * @param list<Entry> $entries
     */
    public function __construct(
        public array $entries,
        public int $page,
    ) {
    }
}

$empty = new EntryPage([], 1);
$populated = new EntryPage(entries: [new Entry()], page: 2);
