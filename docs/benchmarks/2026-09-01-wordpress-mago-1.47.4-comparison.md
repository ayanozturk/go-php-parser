# WordPress Mago 1.47.4 same-machine comparison, 2026-09-01

## Outcome

This is the first current-production WordPress resource comparison in which both tools pass the predeclared 5% process-cold CV gate. The Go pipeline's mean time is 0.569x Mago's and its maximum peak RSS is 0.982x Mago's, passing both the 1.5x mean / 1.25x RSS targets and the equal-or-faster time stretch target.

This is not semantic parity. PHP Strom emits 22,387 diagnostics under every currently registered rule while Mago emits 218,741 findings under its broader strict configuration. The result establishes a stable performance envelope for the current production workload; it does not claim equivalent diagnostic breadth or that the tools report the same findings.

## Reproducible source and workload

- Production parser implementation: `41dfb5580c97f1e448b836ae65077876accb0c64`; the benchmark binary was built at documentation-only descendant `cd277f0b9ccffdecb2eb6ced3beedae79911e340` with `go build -trimpath -ldflags='-s -w'`.
- Candidate binary SHA-256: `662bf9e1fb1a847d001d2ebbcb2cf0d608901a78bbda6d76f474e4e99cb4168d`.
- Mago: 1.47.4, confirmed as GitHub's latest release on 2026-09-01; local binary SHA-256 `8038793e9e7072c22a0ba5868eae1c682ffe98c1fa38999c000a31ef4887da33`.
- Mago configuration: `benchmark-configs/mago/wordpress-develop.toml`, SHA-256 `90375e5fb9b82fe9d9b783ccd107e69842bb621ad4cf3223e20d77052593a17b`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`; dependency fingerprint `be6b7bd587d566ca1308c99b883a937efe3d11b204977499886800af9cc159c4`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`. No rules were disabled and `vendor` was not skipped.
- Machine: Apple M1, 8 cores, 8 GB RAM; macOS 26.5.2 build 25F84; Go 1.27.0; eight Go workers and eight Mago threads.
- Protocol: one unmeasured validation and warmup per tool, ten measured rounds, alternating tool-first order by round, 250ms settle between processes, operating-system file cache warm.

Generated raw JSON remains at `/tmp/phpstrom-mago-comparison.json` and is not committed.

## Accounting

Every Go validation, warmup, and measured run discovered and parsed 5,357/5,357 PHP files with zero failures, 1,451,208 LOC, 47,344,277 bytes, and 22,387 diagnostics.

Mago's `list-files` reports all 3,183 configured primary-source PHP files. The checked-in configuration supplies the remaining 2,174 PHP files through `vendor` as dependency includes, for a fully enumerated 5,357-file configured workload. Mago completed every run and emitted the same 218,741 findings each time: 143,188 errors, 72,386 warnings, 3,129 help findings, and 38 notes. Mago does not expose a parsed/failed count for dependency includes, so that narrower internal accounting remains unavailable.

## Accepted process-cold resource comparison

| Tool | Mean | Median | Min | Max | CV | Max peak RSS |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| go-php-parser production pipeline | 2.059s | 2.063s | 1.948s | 2.142s | 2.61% | 1,061,371,904 bytes |
| Mago 1.47.4 | 3.620s | 3.623s | 3.393s | 3.844s | 3.64% | 1,081,245,696 bytes |

| Gate | Result |
| --- | --- |
| Both full-sample CVs at most 5% | Pass |
| Go mean at most 1.5x Mago | Pass at 0.569x |
| Go maximum peak RSS at most 1.25x Mago | Pass at 0.982x |
| Stretch target: Go mean equal to or faster than Mago | Pass |
| Equivalent semantic coverage | Not established |

## Decision

Accept the timing and resource ratios for the explicitly recorded current-production workloads. Keep the semantic qualification prominent: performance headroom now exists, but diagnostic breadth remains materially different. Continue from a new production profile only when further performance work is justified; do not reopen cleared allocation sites without evidence or trade away the complete registered rule set.
