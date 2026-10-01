<?php

namespace Source\Dto {
    class ImportedItem {}
}

namespace App {
    use Source\Dto\ImportedItem as ItemAlias;

    /**
     * @phpstan-type ItemList list<ItemAlias>
     * @phpstan-type ItemIndex array<string, ItemList>
     * @phpstan-type ItemFactory callable(ItemAlias): ItemIndex
     */
    class LocalTypes
    {
        /** @param ItemIndex $items */
        public function inspectIndex(array $items): void {}

        /** @param ItemFactory $factory */
        public function inspectFactory(callable $factory): void {}
    }
}
