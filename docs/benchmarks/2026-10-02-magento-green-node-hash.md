# Magento green-node interning, 2026-10-02

## Result

Composite syntax nodes now use a 64-bit structural hash for interner lookup,
then compare the node kind and child pointers before reusing a node. Hash
collisions are retained in a lazily allocated bucket and checked exactly. This
removes the long hexadecimal-pointer string key that was built for every
composite node.

On the pinned Magento corpus, the interleaved process-cold mean fell from
16.424 s to 16.145 s (1.7%). Both ten-run samples passed the 5% CV gate. The
candidate's maximum RSS was 1,046.2 MiB versus 1,049.1 MiB for the baseline.
Every run accounted for 25,390/25,390 files and 285,727 diagnostics.

## Reproduction

- Baseline: commit `2a6d9704`; candidate: this working-tree change.
- Corpus: `test_projects/magento2`, pinned at
  `755e34dd689021c5165db9d35ecff74f7dc51527`.
- Host: Linux amd64, Go 1.27.1, 16 logical CPUs; four workers and
  `GOMAXPROCS=4`.
- Both binaries were built with
  `go build -trimpath -ldflags='-s -w' ./cmd/benchmark`.
- Protocol: one validation and warmup per engine, ten interleaved process-cold
  runs, 250 ms settle delay, no extra runs, warm loop skipped.
- Baseline mean/median/CV/max RSS: 16.424 s / 16.371 s / 0.78% / 1,049.1 MiB.
- Candidate mean/median/CV/max RSS: 16.145 s / 16.119 s / 0.74% / 1,046.2 MiB.

One-iteration allocation profiles recorded 16.51 GB allocated for baseline
and 16.31 GB for candidate. `Interner.Node`'s flat sampled allocation fell
from 1.59 GB to 1.06 GB. These sampled totals are supporting evidence; the
process-cold benchmark is the timing gate.

## Validation

- `SYNTAX_CORPUS_DIR="$PWD/test_projects/magento2" go test ./syntax -count=1 -timeout 30m`:
  passed, including full Magento parse/print identity.
- `GOWORK=off go test ./...`: all packages passed except the known level-5
  differential aggregate, which still reports the two
  `disjunctive-and-range-joins` engine mismatches already present before this
  change.

This is a single-corpus result. It does not update the Symfony, WordPress, or
PSL baseline and is not a four-corpus performance claim.

## Follow-on: fuse composite-node construction

The next iteration computes composite width, content start, and the interner
hash in one child pass instead of separate passes. On the same pinned Magento
corpus, an interleaved ten-run comparison against the preceding working-tree
version measured 15.695 s baseline mean and 15.624 s candidate mean (0.45%
lower). Both samples passed the 5% CV gate (0.30% baseline, 0.63% candidate).
Every run accounted for 25,390/25,390 files and 285,727 diagnostics.

This is a small single-run-set improvement, close to the observed run
variation. Maximum sampled RSS was 1,113.5 MiB baseline and 1,135.8 MiB
candidate, so this iteration does not claim an RSS improvement. The attempted
child-slice ownership fast path was dropped after its paired run showed no
material mean-time change.

Validation for this iteration: `go test ./syntax -count=1` and the Magento
`TestSyntaxCorpusIdentityGate` passed. `GOWORK=off go test ./...` still fails
only at `cmd/diagnostic-diff/TestCheckedInEngineDifferentialBaseline`, which
reports the existing two level-5 `disjunctive-and-range-joins` mismatches.
