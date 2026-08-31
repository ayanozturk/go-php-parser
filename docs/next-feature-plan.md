# Near-term CLI and adoption plan

The primary project target is [Full Static Analyser and Mago-Class Performance Target](full-static-analyser-target.md). Ranked next work lives there. This file is only the remaining CLI/adoption backlog; it must not pull implementation away from Mago-class performance.

## Already in place

- `analyze` uses one immutable project snapshot, the same registered engine as PHP Strom, deterministic diagnostics, file accounting, and stable exit codes.
- `style` remains the compatibility command.
- `list-files` prints the files selected by config.
- Config still accepts `path`, `extensions`, `ignore`, `rules`, and `overrides`.
- Incremental project indexing and a disk analysis cache exist for `analyze`; they are not a substitute for PHP Strom's overlay/scheduling layer.

## Still open (lower priority than analyser coverage)

- Split remaining CLI verbs without breaking `style`: `lint` (style + lint), `format` (registered fixers only), `config` (effective resolved config), `guard` (namespace/`use` dependency rules).
- Config discovery for `go-phpcs.yaml` / `go-phpcs.yml` in addition to `config.yaml`, plus `--config`.
- Shared diagnostic model across parser errors, style, analysis, and guard output.
- Baselines for incremental adoption (`--ignore-baseline`, stale-entry warnings).
- Machine-readable `--format text|json|github|checkstyle` on one schema.
- Conservative formatter: refuse parse errors, apply fixers in stable rule-code order.

## Constraints

- Do not change `make compat-metrics` reporting.
- Treat current config fields as aliases rather than removing them.
- Keep autofix opt-in.
- YAML remains the config format.
- Implementations that touch analysis semantics must follow the analyser target's correctness and benchmark gates, not this file's adoption order.
