# WordPress stability-protocol indicator, 2026-08-31

## Outcome

The process-cold protocol now pins worker `GOMAXPROCS`, discards an unmeasured process-cold warmup, pauses 250ms between subprocesses, and may append extra measured runs when the 5% CV gate fails. The gate still uses every measured sample, including outliers; drop-max CV is recorded only as a diagnostic.

This host indicator is **rejected** as a performance claim: twenty measured WordPress cold runs produced CV 8.22% (drop-max CV 6.27%). File, byte, LOC, parse-failure, and diagnostic accounting were identical on every validation, warmup, and measured run.

## Reproducible source and workload

- Engine commit: `9a06bdb` plus the uncommitted protocol changes recorded with this document's landing commit.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every currently registered rule, eight workers (`GOMAXPROCS=8` in each worker).
- Machine: Apple M1, 8 CPUs; macOS ARM64; Go 1.26.2. The machine was not isolated; this is a protocol indicator, not a quiet-host baseline.
- Command:

```text
go run ./cmd/benchmark --root test_projects/wordpress-develop --paths src,tests,vendor --excludes src/js --cold-runs 10 --warm-iterations 2 --workers 8 --json
```

Defaults applied: `--cold-warmups 1 --settle-ms 250 --extra-cold-runs 10 --max-cv 0.05`.

## Results

All runs discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero parser failures and exactly 24,770 diagnostics.

| Engine | Measured cold runs | Extra runs used | Mean | Median | Min | Max | CV | Drop-max CV | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Candidate `HEAD` | 20 | 10 | 1.552s | 1.504s | 1.368s | 1.930s | 8.22% | 6.27% | Rejected |

The extra-run budget was exhausted. Adding samples reduced neither the full-sample nor the drop-max CV below 5% on this loaded host. Diagnostic accounting did not drift.

## Evidence and decision

Generated JSON remains a local artifact and is not committed. The protocol changes are retained because they record host/runtime pinning, keep the 5% gate honest, and stop a CV rejection from aborting weekly CI artifact upload (`--max-cv 0` on the scheduled job only). No Mago comparison or speed claim follows from these numbers.
