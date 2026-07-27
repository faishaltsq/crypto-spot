# AI Coordination Protocol

## Purpose

Prevent concurrent AI agents from overwriting, staging, or committing each other's work in this shared Git worktree.

## Source of Truth

- `tasks/todo.md`: task state, owner, scope, verification, handoff.
- `.agent-locks/*.lock`: active file ownership. Local only; ignored by Git.
- `git status --short` and `git diff`: actual workspace state. These override claims when they disagree.

## Required Startup Check

Run before claiming work:

```bash
git status --short
git diff
git diff --staged
```

Read `tasks/todo.md` plus every active `.agent-locks/*.lock`. Preserve all changes not clearly created by current agent.

## Claiming Work

Use one task per agent. Task may list at most five target files unless supervisor explicitly approves broader scope.

Add this block under claimed task in `tasks/todo.md`:

```markdown
  - Owner: `<agent-id>`
  - Started (UTC): `YYYY-MM-DDTHH:MM:SSZ`
  - Files: `path/one`, `path/two`
  - Verify: `<exact command>`
  - State: `in_progress`
```

Create `.agent-locks/<task-id>.lock`:

```text
owner=<agent-id>
started_utc=YYYY-MM-DDTHH:MM:SSZ
heartbeat_utc=YYYY-MM-DDTHH:MM:SSZ
files=path/one,path/two
verify=<exact command>
```

Refresh `heartbeat_utc` before long operations or every 30 minutes. Lock file is a coordination signal, not a security boundary.

## Shared Files

Files such as `tasks/todo.md`, `tasks/plan.md`, API contracts, migrations, route registration, and config may be shared. Agent must claim an exclusive edit window in task entry before changing a shared file. Other agents wait, use a separate file, or request supervisor resolution.

## Conflict Procedure

1. Stop edits when active lock overlaps target file or unexpected diff appears.
2. Compare current diff with task claim.
3. Record conflict in `tasks/todo.md`: file, owners, observed change, requested decision.
4. Supervisor selects one owner, sequences edits, or splits interface contract from implementation.
5. Do not use `git reset`, `git checkout`, forced operations, or blanket formatting to resolve conflict.

## Handoff

After verification, replace task state with `done` or `blocked`. Add UTC completion time, exact command result, modified files, and next owner if needed. Remove only lock owned by completing agent.

Blocked task keeps its lock until supervisor reassigns it. Reassignment requires supervisor to inspect Git diff and mark prior claim `abandoned` or `transferred`.

## Staging and Commit Rules

- Never run `git add -A` or `git add .` in shared workspace.
- Stage explicit owned paths only after reviewing `git diff --staged`.
- Never commit files owned by another active task.
- Commit only when user or supervisor requests it.

## Supervisor Audit

Before approving integration, inspect:

```bash
git status --short
git diff --check
git diff
git diff --staged
```

Confirm every changed file has one of: completed task provenance, active owner, or documented pre-existing state. Confirm task verification passed. Escalate ambiguous ownership instead of guessing.
