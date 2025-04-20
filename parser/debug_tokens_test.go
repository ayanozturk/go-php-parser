package parser

import "testing"

func TestDebugTokensForDoublePipe(t *testing.T) {
	php := `<?php
trait Mixin {
    public function nullOrString($value, $message = ''): bool
    {
        null == $value || $message == '';
    }
}`
	DebugPrintTokens(php)
}
