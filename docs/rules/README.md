# Rule documentation

[Back to the project README](../../README.md)

This guide describes each rule currently registered by the tool, one at a time. Each entry explains the check, why it helps, and a small PHP example. Examples show representative supported behavior; they do not claim full PHPStan or PSR coverage.

## Static analysis rules

Analysis rules run with `analyze`. Levelled rules run cumulatively up to the configured `analysis_level`; unlevelled rules run when that setting is omitted.

| Page | Registered rules |
| --- | ---: |
| [Level 0](level-0.md) | 2 |
| [Level 1](level-1.md) | 2 |
| [Level 2](level-2.md) | 17 |
| [Level 3](level-3.md) | 5 |
| [Level 4](level-4.md) | 1 |
| [Level 5](level-5.md) | 1 |
| [Level 6](level-6.md) | 5 |
| [Level 7](level-7.md) | 1 |
| [Level 8](level-8.md) | 1 |
| [Level 9](level-9.md) | 1 |
| [Level 10](level-10.md) | 0 |
| [Unlevelled rules](unlevelled.md) | 4 |

These counts are registered engine rules, not diagnostic identifiers. One analysis rule may emit multiple diagnostic codes. The level pages describe the registered rule entries and summarize tested coverage and known boundaries.

## Style rules

The style checker runs with the default CLI command and supports per-rule configuration. The [style rule page](style.md) documents all 16 registered style rules.

## Updating this guide

When adding, removing, renaming, or changing a rule, update its entry and any coverage/boundary text on the matching page in the same change. The documentation tests compare rule headings and required example/rationale sections with the live registries. See [AGENTS.md](../../AGENTS.md) for the repository workflow.
