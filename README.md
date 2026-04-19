# asm — Agent Skills Manager

A CLI tool that manages a centralized Git repository of AI agent skills and keeps them in sync across machines and coding agents (Claude Code, Cursor, Codex).

## How it works

Rather than copying files, `skills` uses symlinks:

1. Your skills live in a single Git repository (default: `~/.skills-repo`)
2. Each agent's global skills folder is replaced with a symlink to that repo's `skills/` directory
3. All agents on the machine share the same skills instantly
4. Syncing across machines is done via `git pull`/`git push`

```
~/.claude/skills  ->  ~/.skills-repo/skills/
~/.cursor/skills  ->  ~/.skills-repo/skills/
~/.codex/skills   ->  ~/.skills-repo/skills/
```

## Installation

```bash
# Build and install to /usr/local/bin/skills
make install
# or with sudo if needed
sudo make install
```

## Quick start

```bash
skills init    # First-time setup — interactive wizard
skills status  # Verify everything is working
skills sync    # Pull latest and push local changes
```

## Commands

| Command | Description |
|---|---|
| `skills init` | First-time setup: clone/create/point to a repo, migrate existing skills, create symlinks, optionally install cron |
| `skills sync` | Commit local changes, pull with rebase, push, verify symlinks |
| `skills status` | Show repo path, branch, ahead/behind, uncommitted changes, skill count, and symlink state per agent |
| `skills list` | List all skills with their descriptions |
| `skills add <path>` | Copy a local skill folder into the central repo and commit |
| `skills remove <name>` | Remove a skill from the central repo and commit |
| `skills link` | (Re)create symlinks for all detected agents |
| `skills unlink` | Remove symlinks and restore any backed-up original folders |
| `skills cron doctor [--fix]` | Check cron health and optionally repair stale `skills` binary paths |

## Skill format

Skills follow the [agentskills.io](https://agentskills.io) open standard, compatible across all supported agents:

```
skill-name/
├── SKILL.md          # Required — YAML frontmatter + instructions
├── scripts/          # Optional — executable scripts
├── references/       # Optional — documentation loaded into context
└── assets/           # Optional — templates, icons, output files
```

`SKILL.md` must include YAML frontmatter:

```yaml
---
name: skill-name
description: "When to use this skill and what it does"
---
```

## Supported agents

| Agent | Global skills path |
|---|---|
| Claude Code | `~/.claude/skills/` |
| Cursor | `~/.cursor/skills/` |
| Codex | `~/.codex/skills/` |

Detection is automatic: `skills init` scans for installed agents and only links the ones present.

## Configuration

Config is saved at `~/.config/skills/config.json`:

```json
{
  "repo_path": "~/.skills-repo",
  "remote_url": "git@github.com:you/skills.git",
  "agents": ["claude-code", "cursor", "codex"],
  "sync_on_start": false
}
```

Sync logs are written to `~/.config/skills/sync.log`.

## Automatic sync (cron)

`skills init` offers to install a cron entry that syncs every 30 minutes using the current `skills` binary path:

```cron
*/30 * * * * /path/to/current/skills sync >> ~/.config/skills/sync.log 2>&1
```

If the `skills` binary moves later, run `skills cron doctor` to check the entry or `skills cron doctor --fix` to repair it.

## Platform support

| Platform | Support |
|---|---|
| macOS | Full (including system notifications on sync conflict) |
| Linux | Full (conflict notification via stderr/log only) |
| Windows | Not supported |
