# Subagent delegation

Proactively delegate bounded, low-risk work to the `worker` subagent when it can complete the task independently and doing so keeps the main agent focused or reduces expensive main-agent usage.

Prefer the `worker` for:

- small, clearly scoped implementation tasks
- repetitive or mechanical edits
- focused test additions or updates
- narrow bug fixes with an obvious validation path
- straightforward refactors that follow existing project patterns
- repository searches, call-site discovery, and other bounded investigation supporting an implementation task

Prefer delegation over doing routine implementation in the main agent when the worker can complete it independently.

Keep with the main agent:

- architecture and design decisions
- ambiguous or open-ended investigations
- complex debugging where the root cause is unclear
- cross-cutting changes requiring broad coordination
- release, deployment, or migration work
- security-sensitive or high-risk changes
- final integration and review

Do not run multiple write-capable agents on overlapping files or overlapping areas of responsibility.

When delegating:

- give the worker a narrow objective and clear acceptance criteria
- identify relevant files or constraints when known
- avoid delegating unnecessary surrounding context
- keep ownership of architectural decisions with the main agent

The main agent remains responsible for:

- reviewing all worker changes
- resolving integration issues
- checking that scope was not widened unnecessarily
- running appropriate project-level validation
- confirming the final implementation is coherent before completion

# Commit and push authorization

The user has explicitly authorized committing and pushing completed, in-scope changes for this project.

This authorization does not broaden task scope.

Before committing or pushing:

- review the final diff
- run relevant tests, linting, type checks, or other appropriate validation
- avoid including unrelated changes
- use a concise commit message that accurately describes the completed work

Do not commit or push incomplete, unvalidated, or out-of-scope changes.
