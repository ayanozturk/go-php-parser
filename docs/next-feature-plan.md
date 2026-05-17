# Next Feature Plan For go-php-parser

## Summary

This roadmap focuses on the next practical features for `go-php-parser`: a fast PHP parser with style rules, early analysis rules, autofix support, config, reporting, and compatibility metrics.

The goal is to make the tool easier to adopt in real PHP codebases while preserving the current command behavior and configuration shape for existing users.

## Key Changes

- Split the CLI into clearer tools while keeping the current `style` command supported:
  - `style` remains available for backwards compatibility.
  - `lint` runs style and lint rules.
  - `analyze` runs analysis rules.
  - `format` applies deterministic autofixes only.
  - `list-files` prints the exact files selected by config.
  - `config` prints the effective resolved configuration.
  - `guard` checks lightweight architectural dependency rules.
- Extend configuration without breaking current `config.yaml` fields:
  - Keep accepting `path`, `extensions`, `ignore`, `rules`, and `overrides`.
  - Add source path, include, exclude, and extension settings.
  - Add separate lint, analyze, format, and guard sections.
  - Add config discovery for `go-phpcs.yaml`, `go-phpcs.yml`, then `config.yaml`.
  - Add a `--config` flag to override discovery.
- Add a shared diagnostic model for parser errors, style issues, analysis issues, and guard violations.
- Add baseline support so teams can adopt checks incrementally:
  - Generate lint and analysis baselines.
  - Ignore known existing issues by default when a baseline is configured.
  - Allow `--ignore-baseline` to report all issues.
  - Warn when baseline entries become stale.
- Add machine-readable reporting:
  - Keep the current human-readable output as the default.
  - Add `--format text|json|github|checkstyle`.
  - Use one diagnostic schema for all non-text formats.
- Add a conservative formatter workflow:
  - Apply only existing registered fixers at first.
  - Run fixes in stable rule-code order.
  - Refuse to format files with parse errors.
- Add a lightweight architectural guard:
  - Read namespace and `use` statements from the existing AST.
  - Support path glob and namespace prefix matching.
  - Report denied dependencies as normal diagnostics.

## Implementation Notes

- Refactor command execution so each file is parsed once and then passed to the selected tool.
- Keep `style` as the default command until a later major cleanup.
- Avoid changing compatibility metrics; `make compat-metrics` should continue to work unchanged.
- Keep autofix behavior opt-in and conservative.
- Make effective config output deterministic so it can be used in tests and debugging.
- Treat current config fields as compatibility aliases for the new config model rather than removing them.

## Test Plan

- Add unit tests for config discovery and migration from current `config.yaml` fields into the new effective config.
- Add output tests for text, JSON, GitHub, and Checkstyle diagnostics.
- Add baseline tests for generation, matching, stale entries, and `--ignore-baseline`.
- Add CLI tests proving `style` remains supported and matches the intended lint behavior.
- Add guard tests with small PHP fixtures for allowed and denied namespace imports.
- Run `go test ./...`.
- Run `make compat-metrics` to confirm parser compatibility reporting still works.

## Assumptions

- The first implementation should prioritize adoption features over deeper type inference.
- YAML remains the primary config format.
- Existing users should not need to change their current `config.yaml` immediately.
- Formatter support should initially reuse the current fixer registry instead of adding a separate formatting engine.
- The architectural guard should start with namespace and import rules before adding deeper dependency analysis.
