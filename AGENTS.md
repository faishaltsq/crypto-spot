# AI Coordination Rules

All AI agents working in this repository must coordinate through `docs/AI_COORDINATION.md`.

Before editing:

1. Read `git status --short`, `git diff`, `tasks/todo.md`, and active `.agent-locks/*.lock` files.
2. Claim one bounded task in `tasks/todo.md` with agent ID, UTC start time, files, and verification command.
3. Create `.agent-locks/<task-id>.lock` using the documented format.
4. Do not edit files claimed by another active lock. Ask the supervisor or task owner to resolve shared-file changes.

While editing:

- Keep task scope bounded. Do not reformat, revert, stage, commit, or delete other agents' work.
- Re-check `git status --short` before each edit to shared files.
- Stop when another agent changes a locked file. Record conflict; do not overwrite it.
- Do not add credentials, tokens, API keys, or `.env` contents to tracked files.

Before handoff:

1. Run task verification command.
2. Record result, changed files, and handoff note in `tasks/todo.md`.
3. Remove only own lock after verification or explicit handoff.
4. Report pre-existing worktree changes separately from own changes.

Supervisor authority:

- Resolve ownership conflicts.
- Reassign abandoned locks after checking agent heartbeat and Git diff.
- Block merges or commits when task ownership, verification, or working-tree provenance is unclear.
