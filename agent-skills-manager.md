# Agent Skills Manager — Implementation Plan

## Overview

**Agent Skills Manager** (`skills`) is a macOS-first CLI tool written in Go that manages a centralized Git repository of AI agent skills, syncing them across machines and coding agents (Claude Code, Cursor, Codex). It replaces per-agent, per-machine skill management with a single source of truth.

---

## Background & Motivation

Coding agents like Claude Code, Cursor, and Codex all support "skills" — folders containing a `SKILL.md` file (with YAML frontmatter and markdown instructions) plus optional supporting files like `scripts/`, `references/`, `templates/`, and `assets/`. All three agents follow the [agentskills.io](https://agentskills.io) open standard, meaning the folder format is compatible across agents.

The problem: skills are currently siloed per agent and per machine. Editing a skill for one agent doesn't propagate to others, and keeping multiple machines in sync requires manual effort.

The solution: a single Git repository stores all global skills. On each machine, each agent's global skills folder is replaced with a symlink pointing to a local clone of that repository. A CLI tool handles cloning, symlinking, and syncing.

---

## Core Concepts

### Skill Format (agentskills.io standard)

All supported agents use the same structure:

```
skill-name/
├── SKILL.md          # Required. YAML frontmatter + markdown instructions
├── scripts/          # Optional. Executable scripts (Python, Bash, etc.)
├── references/       # Optional. Documentation loaded into context as needed
├── assets/           # Optional. Templates, icons, output files
└── agents/           # Optional. Agent-specific UI metadata (e.g. openai.yaml)
```

`SKILL.md` must include YAML frontmatter:

```yaml
---
name: skill-name
description: "When to use this skill and what it does"
---
```

### Global vs Project Skills

- **Global skills** live in agent-specific user directories (e.g. `~/.claude/skills/`, `~/.cursor/skills/`, `~/.codex/skills/`). These are managed by `skills`.
- **Project skills** live inside the project directory (e.g. `.claude/skills/`, `.cursor/skills/`). These are project-specific and intentionally left untouched by `skills`.

### The Sync Mechanism

Rather than copying files, `skills` uses **symlinks**:

1. The Git repository is cloned to a local path (default: `~/.skills-repo`)
2. Each agent's global skills folder is replaced with a symlink pointing to the cloned repo's `skills/` directory
3. Since all agents point to the same directory, editing any skill is immediately reflected across all agents on that machine
4. Syncing across machines is done via Git (pull/push)

Why symlinks over hard links? Hard links don't work across filesystems or network shares, and they can't point to directories. Symlinks are transparent to agents (they follow them naturally), easy to debug, and easy to remove.

---

## Agent-Specific Paths

| Agent | Global Skills Path |
|---|---|
| Claude Code | `~/.claude/skills/` |
| Cursor | `~/.cursor/skills/` |
| Codex | `~/.codex/skills/` |

The tool auto-detects which agents are installed by checking for these directories.

---

## CLI Tool Design

### Name

- **Full name:** Agent Skills Manager
- **Primary command:** `skills`
- **Fallback command (if `skills` is taken):** `skills-sync` or `asm`

On install, the tool checks if `skills` conflicts with an existing binary. If so, it installs under the fallback name and informs the user.

### Commands

```
skills init           # First-time setup on a machine
skills sync           # Pull from Git, sync symlinks
skills status         # Show current sync status and agent link state
skills list           # List all skills in the central repo
skills add <path>     # Add a local skill folder to the central repo
skills remove <name>  # Remove a skill from the central repo
skills link           # (Re)create symlinks for all detected agents
skills unlink         # Remove symlinks and restore original folders
```

---

## `skills init` — Onboarding Flow

This is the most important command. It handles the initial setup on a machine.

### Step 1: Detect existing agent skill folders

Scan known paths for each supported agent. For each one found, record:
- Whether the path exists
- Whether it is already a symlink (i.e. already managed by `skills`)
- How many skills it contains

### Step 2: Ask for repository source

Present the user with three options:

**Option A — Use an existing local Git repo**
- Prompt for path
- Validate it is a Git repo with the expected structure

**Option B — Clone from a remote Git repo**
- Prompt for URL
- Clone to the configured local path (default: `~/.skills-repo`)

**Option C — Create a new repo**
- Scaffold a new directory with the correct structure
- Initialize a Git repo
- Print instructions for pushing to GitHub manually (do not automate GitHub auth)

### Step 3: Handle existing skills

For each detected agent that has an existing (non-symlinked) skills folder:
- Show the user what was found
- Ask: migrate existing skills into the central repo, or discard them?
- If migrating: copy skills into the central repo, commit with message `feat: migrate skills from <agent> on <hostname>`
- Back up the original folder to `<path>.backup` before replacing

### Step 4: Create symlinks

Replace each agent's skills folder with a symlink to the central repo's `skills/` directory:

```
~/.claude/skills -> ~/.skills-repo/skills/
~/.cursor/skills -> ~/.skills-repo/skills/
~/.codex/skills  -> ~/.skills-repo/skills/
```

### Step 5: Write config

Save configuration to `~/.config/skills/config.json`:

```json
{
  "repo_path": "~/.skills-repo",
  "remote_url": "git@github.com:username/skills.git",
  "agents": ["claude-code", "cursor", "codex"],
  "sync_on_start": true
}
```

---

## `skills sync` — Sync Logic

This is the core command, designed to be safe to run frequently (including via cron).

### Flow

1. `git fetch` from remote
2. Check for local uncommitted changes
3. If local changes exist: `git add -A && git commit -m "sync: update skills on <hostname> at <timestamp>"`
4. `git pull --rebase` (rebase is cleaner than merge for this use case)
5. If conflict detected: abort rebase, notify user (see Conflict Handling), exit with non-zero code
6. If no conflict: `git push`
7. Verify all agent symlinks are still intact (re-create any that are broken)
8. Print summary: skills added/removed/modified, push/pull status

### No-op behavior

If there is nothing to commit and the remote is already up to date, the command exits cleanly with no output (safe for cron).

### Conflict Handling

Merge conflicts are expected to be rare (single user, all changes go through the same person). If a conflict does occur:

1. Abort the rebase (`git rebase --abort`)
2. Log the error to `~/.config/skills/sync.log`
3. Send a macOS system notification via `osascript`:

```
osascript -e 'display notification "Skills sync conflict — manual resolution needed" with title "Agent Skills Manager" subtitle "Run: skills sync --resolve"'
```

4. Exit with a non-zero code so cron knows it failed

The user then resolves the conflict manually and runs `skills sync` again.

---

## Cron Integration

`skills init` offers to install a cron entry automatically using the current `skills` binary path:

```cron
*/30 * * * * /path/to/current/skills sync >> ~/.config/skills/sync.log 2>&1
```

The user can adjust the interval or disable it at any time, and `skills cron doctor --fix` can repair stale paths if the binary later moves.

---

## Repository Structure

The central Git repository has the following layout:

```
skills-repo/
├── README.md
├── .gitignore
└── skills/
    ├── skill-one/
    │   └── SKILL.md
    ├── skill-two/
    │   ├── SKILL.md
    │   └── scripts/
    │       └── run.py
    └── ...
```

All skills live flat inside `skills/`. No per-agent subdirectories — the format is shared across all agents via the agentskills.io standard.

---

## Tech Stack

| Concern | Choice | Rationale |
|---|---|---|
| Language | Go | Single compiled binary, no runtime deps, easy distribution, good agent codegen support |
| Config format | JSON | Simple, human-readable, easy to edit manually |
| Git operations | Shell via `os/exec` | Simpler than embedding a Go Git library; `git` is always available on target machines |
| Notifications | `osascript` | Native macOS notifications, no deps |
| Testing | Go standard `testing` package | Sufficient for CLI unit tests |

---

## Distribution

### Phase 1 — GitHub Releases (initial)

- Build binary with `go build -o skills ./cmd/skills`
- Cross-compile for `darwin/amd64` and `darwin/arm64` (Apple Silicon)
- Tag release: `git tag v1.0.0 && git push --tags`
- GitHub Actions workflow automatically builds binaries and creates a release on tag push
- User downloads binary and adds to PATH manually, or via install script

**Install script (one-liner):**
```bash
curl -fsSL https://raw.githubusercontent.com/username/skills/main/install.sh | bash
```

### Phase 2 — Homebrew Tap (once stable)

Create a Homebrew tap repository (`homebrew-skills`). On each release:
1. Build and upload binaries to GitHub release
2. Calculate SHA256 of each binary
3. Update the Homebrew formula with new version and hashes
4. Push formula to tap repo

Users install with:
```bash
brew tap username/skills
brew install skills
```

Updating is then just `brew upgrade skills`.

---

## Platform Support

| Platform | Support Level |
|---|---|
| macOS (primary) | Full support |
| Linux | Best-effort (symlinks work, no `osascript` notifications) |
| Windows | Out of scope initially |

On Linux, conflict notifications fall back to stderr output and log file only.

---

## Error Handling & Edge Cases

- **Symlink already exists:** check target matches expected repo path; warn if not
- **Agent not installed:** skip silently, log at debug level
- **Repo not initialized:** print friendly error pointing to `skills init`
- **Git not installed:** exit with clear error message
- **No internet / remote unreachable:** skip push/pull, continue with local operations
- **Skill name conflict during migration:** prompt user to rename or skip
- **`skills` binary name conflict on install:** detect, fall back to `skills-sync` or `asm`, inform user

---

## Out of Scope (for now)

- Project-level skill management (`.claude/skills/`, `.cursor/skills/`, etc.)
- Windows support
- Menu bar / tray icon status indicator (nice-to-have, post-MVP)
- Explicit skill versioning (Git history provides implicit versioning)
- Multi-user or team skill sharing (single-user tool)
- NAS / network share integration
- Automatic GitHub repo creation during `init`

---

## Future Ideas

- `skills doctor` command to diagnose broken symlinks, missing agents, stale config
- Menu bar app (Swift/SwiftBar) showing sync status with green/red icon
- `skills search <query>` to find skills by name or description
- Skill validation: check `SKILL.md` frontmatter for required fields before committing
- Support for private skill registries (install skills from a URL like Codex does)
