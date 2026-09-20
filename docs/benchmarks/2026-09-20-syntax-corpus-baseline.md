# Syntax corpus baseline — 2026-09-20

This record closes step 1 of `plan.MD`: capture the exact lossless-syntax baseline before isolated recovery work or performance comparison.

## Environment

- Parser commit: `1cf69b079c40e0b408da771102bdb63b9393da85`
- Worktree before validation: clean; local `main` matched `origin/main`
- Go: `go1.27.0 darwin/arm64`
- Host: Apple M1 MacBook Pro, 8 cores (4 performance and 4 efficiency), 8 GB memory
- OS/kernel: Darwin 25.5.0, arm64
- Symfony revision: `ae256f91a9cacc470fe77eca87aedd81c65ca55e`
- WordPress revision: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`

## Focused validation

```text
GOWORK=off go test ./syntax ./lexer ./analyse -count=1
ok github.com/ayanozturk/go-php-parser/syntax  1.254s
ok github.com/ayanozturk/go-php-parser/lexer   2.712s
ok github.com/ayanozturk/go-php-parser/analyse 5.954s
```

## Corpus metrics

| Corpus | Files discovered | Passing | Failing | Identity | Bytes | Tokens | Tokens/KB | Wall time |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Symfony | 10,474 | 10,474 | 0 | 100% | 83,521,359 | 8,088,964 | 99.1734 | 3.841s |
| WordPress | 5,362 | 5,362 | 0 | 100% | 47,359,328 | 5,005,113 | 108.2202 | 2.552s |

Commands:

```bash
GOWORK=off go run ./cmd/syntax-metrics --root "$PWD/test_projects/symfony" --json
GOWORK=off go run ./cmd/syntax-metrics --root "$PWD/test_projects/wordpress-develop" --json
```

The raw JSON reports are retained outside the repository at:

- `/tmp/go-php-parser-syntax-metrics-symfony-1cf69b07.json`
- `/tmp/go-php-parser-syntax-metrics-wordpress-1cf69b07.json`

## Interpretation

Both corpora have complete discovered/passing file accounting, zero syntax-metrics failures, and 100% source identity at this commit. This is a correctness baseline, not a process-cold performance baseline and not evidence of static-analysis diagnostic parity. The dedicated identity-gate commands, recovery coverage audit, cancellation/robustness validation, and stable split parser benchmarks remain open in `plan.MD`.
