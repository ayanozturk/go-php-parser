# Subagent delegation

Proactively delegate small, clearly scoped, repetitive implementation tasks to the `worker` subagent when doing so would keep the main agent focused or reduce cost. The worker is suitable for bounded mechanical edits, focused test additions, and narrow fixes with straightforward validation.

Keep architectural decisions, ambiguous investigations, release work, and changes requiring broad coordination with the main agent. Do not run multiple write-capable agents on overlapping files. The main agent remains responsible for reviewing the worker's changes, running appropriate project-level validation, and reporting the final result.

# Commit and push authorization

The user has explicitly authorized committing and pushing completed, in-scope changes for this project. This authorization does not broaden task scope; verify changes and relevant validation before committing or pushing.
