<?php

namespace Composer {
    final class InstalledVersions
    {
        public static function getPrettyVersion(string $packageName): ?string
        {
            return null;
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
