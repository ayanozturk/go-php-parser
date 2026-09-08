# PHPStan-compatible rules: level 8

<!-- rule-inventory: level=8 introduced=1 cumulative=32 -->

[Back to the README static-analysis section](../../README.md#static-analysis) · [Open the analyser capability matrix](../analyser-capability-matrix.md)

## Rule inventory

- **Introduced at this level:** 1 registered levelled rule, `Level8.MethodNonObject`.
- **Cumulative registered levelled rules:** 32.
- **Checked-in differential pack:** 35 cases in `testdata/diagnostic-differential-level8`.

## Coverage and boundaries

`Level8.MethodNonObject` reports selected method calls where a nullable or otherwise non-object receiver can reach the call. It complements the lower-level non-object checks with the level-8 nullable-object boundary and uses inferred receiver types where available.

The rule does not model every dynamic call, magic method, or unresolved union. The `known-method-negated-instanceof-or` clean control proves a known method on the right of `!$x instanceof T || …` after false-scope narrowing. True-scope `method_exists`/`is_object` on property-fetch receivers (`$this->prop`, `$obj->prop`) via `methodReceiverGuardKey` keeps guarded calls clean in `if`, ternary, and `&&` forms; proofs live on function-scope context keys. Additional clean controls replay a boolean variable assigned from a property null-check, PHPUnit `assertSame` on `$this->prop`, assignment-in-condition nullable narrowing, and `getPrevious()` after a constructor previous-exception argument. Levels 0 through 7 are cumulative.
