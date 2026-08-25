# WordPress post-allocation rebaseline, 2026-08-25

## Outcome

The allocation changes are directionally strong but this rebaseline is rejected as a performance claim. Candidate and baseline produced identical file, byte, LOC, parse-failure, and diagnostic accounting in every validation and measured run. The candidate was approximately 23% faster by both cold mean and median in two interleaved attempts, and its maximum cold peak RSS was approximately 5% lower. Neither attempt met the required 5% coefficient-of-variation gate for both engines, so the roadmap stability gate remains open.

## Reproducible source and workload

- Candidate commit: `6ac1a4024aa9f1099abe81879da56c4124e89314`.
- Previous-engine baseline commit: `b802a769b658cf6cc2290f6cde1b6bacc377951a`.
- WordPress commit: `daaca56d3d6a9a42a0c87f6eda766c33a77c1d05`.
- Composer dependency fingerprint: `be6b7bd587d566ca1308c99b883a937efe3d11b204977499886800af9cc159c4` (`vendor/composer/installed.json`, SHA-256).
- Paths: `src`, `tests`, `vendor`; excluded path: `src/js`.
- Engine configuration: every currently registered rule, eight workers.
- Machine: Apple M1, 8 CPUs, 8 GiB memory; macOS ARM64; Go 1.26.2.
- Candidate and baseline were built with `go build -trimpath -ldflags='-s -w' ./cmd/benchmark`.
- The candidate harness ran one unmeasured full-pipeline validation for each engine, then alternated candidate-first and baseline-first order by round.

The effective command was:

```text
benchmark --root test_projects/wordpress-develop --paths src,tests,vendor --excludes src/js --cold-runs <10-or-20> --warm-iterations 11 --workers 8 --baseline-binary <baseline> --max-cv 0.05 --json
```

## Results

All runs discovered and parsed 5,357/5,357 files, covering 1,451,208 LOC and 47,344,277 bytes with zero parser failures and exactly 30,007 diagnostics.

| Attempt | Engine | Cold runs | Mean | Median | Min | Max | CV | Max peak RSS | Gate |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 1 | Candidate `6ac1a40` | 10 | 0.847s | 0.834s | 0.789s | 0.958s | 6.08% | 1,000,390,656 bytes | Rejected |
| 1 | Baseline `b802a76` | 10 | 1.101s | 1.101s | 1.012s | 1.196s | 5.21% | 1,055,064,064 bytes | Rejected |
| 2 | Candidate `6ac1a40` | 20 | 0.853s | 0.844s | 0.776s | 0.979s | 6.94% | 1,012,908,032 bytes | Rejected |
| 2 | Baseline `b802a76` | 20 | 1.110s | 1.104s | 1.013s | 1.243s | 5.18% | 1,072,005,120 bytes | Rejected |

In attempt 1 the candidate mean was 23.1% lower, the median was 24.2% lower, and maximum peak RSS was 5.2% lower than the baseline. In attempt 2 the candidate mean was 23.1% lower, the median was 23.5% lower, and maximum peak RSS was 5.5% lower. The repeatability of the direction is useful engineering evidence, but the CV failures prevent accepting either result as the roadmap performance baseline.

The candidate warm-loop means were 0.469s for attempt 1 and 0.482s for attempt 2. Warm timing is reported for context only and is not compared with the process-cold baseline binary.

## Evidence and decision

- Checked-in attempt 1 JSON SHA-256: `1b304856136ffcee290288a9d30388e8f84b04f2281ffb36eb3b03fa78038e82`.
- Checked-in attempt 2 JSON SHA-256: `49fdbb7996b5017e258605afd1c774152a040e626084d827def3e8ff8cecc684`.

The immutable method views and copy-on-write branch state are retained: they preserve semantic output and show a repeated material reduction in cold elapsed time without increasing peak RSS. The benchmark-protocol work is also complete and behaved as intended by rejecting both otherwise favorable attempts. The next performance task is to identify and control host/runtime variance or define a more robust predeclared statistical acceptance method before rerunning the gate. No Mago comparison or performance-parity claim follows from these results.
