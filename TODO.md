# Future Work

## Agent Support

- [ ] **Windsurf** — Add agent definition if it adopts the agentskills.io standard
- [ ] **Copilot** — Add agent definition once skill format is supported
- [ ] **Aider** — Investigate skill/convention format
- [ ] **Zed AI** — Investigate skill/convention support
- [ ] **JetBrains AI** — Investigate skill/convention support
- [ ] **Agent-specific metadata** — Use the `agents/` subdirectory in skills to handle per-agent quirks (e.g. different frontmatter fields)

## Skill Management

- [ ] **`skills search <query>`** — Search skills by name, description, or content
- [ ] **`skills doctor`** — Diagnose broken symlinks, missing agents, config issues, stale backups
- [ ] **`skills validate`** — Check SKILL.md frontmatter, required fields, file structure before commit
- [ ] **`skills install <url>`** — Install a skill from a Git URL or skill registry
- [ ] **`skills update`** — Pull latest version of an installed remote skill
- [ ] **Skill templates** — `skills add --template` to scaffold a new skill with boilerplate SKILL.md
- [ ] **Skill dependencies** — Allow skills to declare dependencies on other skills

## Sync & Git

- [ ] **Conflict resolution UI** — Interactive conflict resolution instead of just aborting rebase
- [ ] **Multi-branch support** — Use branches for experimental skills, merge when ready
- [ ] **Selective sync** — Allow per-agent or per-machine skill filtering (not all skills on all machines)
- [ ] **Auto-create GitHub repo** — Offer to create a remote repo during `skills init` via `gh`

## Platform & Distribution

- [ ] **Windows support** — Symlink handling, Task Scheduler instead of cron
- [ ] **Homebrew formula** — `brew install tholst/tap/skills`
- [ ] **Shell completions** — Generate bash/zsh/fish completions via cobra
- [ ] **`skills self-update`** — Update the binary in place

## UX & Observability

- [ ] **Menu bar indicator** — macOS menu bar app (Swift/SwiftBar) showing sync status
- [ ] **`skills diff`** — Show what changed since last sync across all skills
- [ ] **Richer `skills list`** — Show skill metadata (author, tags, last modified)
- [ ] **Dry-run mode** — `skills sync --dry-run` to preview what would happen
- [ ] **Colored output** — Use ANSI colors for status, warnings, errors

## Multi-User / Team

- [ ] **Shared team skill repos** — Support multiple remotes or skill sources
- [ ] **Private skill registries** — Install from authenticated URLs or private repos
- [ ] **Skill namespacing** — Avoid name collisions when combining skills from multiple sources
