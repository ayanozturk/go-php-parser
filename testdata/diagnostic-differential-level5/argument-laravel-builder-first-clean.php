<?php

namespace Illuminate\Database\Eloquent {
    class Model {}

    final class Builder
    {
        public function first(): ?Model
        {
            return random_int(0, 1) === 0 ? new Model() : null;
        }
    }
}

namespace App {
    use Illuminate\Database\Eloquent\Builder;
    use Illuminate\Database\Eloquent\Model;

    function consume(Model $model): void {}

    function run(): void
    {
        $model = (new Builder())->first();
        if ($model === null) {
            return;
        }
        consume($model);
    }
}
