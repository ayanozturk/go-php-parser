<?php

namespace Composer {
    final class InstalledVersions
    {
        public static function getPrettyVersion(string $packageName): ?string
        {
            return random_int(0, 1) === 0 ? null : '1.0.0';
        }
    }
}

namespace App {
    function requireVersion(string $version): void
    {
    }

    function passComposerVersion(): void
    {
        requireVersion(\Composer\InstalledVersions::getPrettyVersion('vendor/package'));
    }
}
