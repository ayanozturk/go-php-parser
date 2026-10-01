<?php

namespace Source\Dto {
    class ImportedItem {}
}

namespace App {
    use Source\Dto\ImportedItem as ItemAlias;

    /** @phpstan-type ItemList list<ItemAlias> */
    class LocalTypes
    {
        /** @param ItemList $items */
        public function incompatible(int $items): void {}

        /** @param ItemListSuffix $value */
        public function aliasNameBoundary($value): void {}
    }
}
