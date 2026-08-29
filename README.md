# Karl AI Agent Dotfiles

Global, portable configuration for a controlled development topology:

```text
USER -> ORCHESTRATOR -> WORKER -> ORCHESTRATOR -> REVIEWER -> ORCHESTRATOR
```

Only ORCHESTRATOR coordinates. WORKER owns the change. REVIEWER evaluates the result independently and does not repair. Repair cycles are bounded at two.

## Distribution

| Content | OpenCode | Codex |
|---|---|---|
| `skills/*/SKILL.md` | Reads `~/.agents/skills` natively | Reads `~/.agents/skills` natively |
| Entry rules | Managed block in `~/.config/opencode/AGENTS.md` | Managed block in `~/.codex/AGENTS.md` |
| Orchestrator | `karl-orchestrator.md`, primary agent | Root session guided by `AGENTS.md` |
| Worker | Markdown agent config | TOML agent config |
| Reviewer | Markdown agent config | TOML with `sandbox_mode = "workspace-write"` |

Skills hold the procedure for each role. Files under `harnesses/` hold the agent boundary: permissions, mode, role, and authority. On OpenCode each agent can only load the skill for its role; ORCHESTRATOR does not load `karl-work` or `karl-review`.

Agent configuration (including model selection) lives in `harnesses/` and is not documented here.

## Install

PowerShell 7 is required on Windows and Linux. On Windows, enable Developer Mode to create symlinks without elevation. On Linux, the installer uses normal symlink support and does not query the Windows registry.

```powershell
pwsh -File "$HOME/.agents/install.ps1" -DryRun
pwsh -File "$HOME/.agents/install.ps1"
```

The installer:

1. Validates `karl-*` skills.
2. Inserts or updates only the block delimited by `karl-ai` comments in each global `AGENTS.md`.
3. Creates per-file symlinks for each harness's custom agents.
4. Keeps backups under `~/.agents-backup/<timestamp>/` before replacing content.

If a custom agent already exists and is not the expected symlink, the installer stops. Use `-Force` to back it up and replace it.

## Uninstall

```powershell
pwsh -File "$HOME/.agents/install.ps1" -Uninstall -DryRun
pwsh -File "$HOME/.agents/install.ps1" -Uninstall
```

Uninstall removes only:

- blocks delimited by `<!-- karl-ai: controlled-development -->`
- symlinks whose target is inside this repository

It does not remove other rules, skills, or agents.

## Sync to Another Machine

```powershell
git clone <private-repository> "$HOME/.agents"
pwsh -File "$HOME/.agents/install.ps1"
```

Installs or updates skills directly inside `~/.agents/skills`; both harnesses discover them with no extra steps.

External skills installed by other managers stay local and are not part of this repository. Git only versions the Karl skills maintained here.

## Skill Diagnostics

OpenCode does not load `~/.agents/skills` when started with:

```text
OPENCODE_DISABLE_EXTERNAL_SKILLS=1
```

The installer warns if that variable is detected. Remove it from the environment before starting OpenCode; no need to duplicate or link skills into `~/.config/opencode/skills`.

## E2E

The offline test runs the full cycle in a temporary home. No credentials, Pester, or other dependencies required:

```powershell
pwsh -NoProfile -File tests/e2e/install-lifecycle.ps1
```

From Windows you can also reproduce the Linux run with Docker:

```powershell
docker run --rm --mount "type=bind,source=$PWD,target=/repo" -w /repo mcr.microsoft.com/powershell:latest pwsh -NoProfile -File tests/e2e/install-lifecycle.ps1
```

The live test requires `OPENCODE_API_KEY`. It uses a temporary home and a temporary git repository, runs `karl-orchestrator` non-interactively, and keeps JSON stdout and stderr under `TestResults/`. Policy is to install `opencode-ai@latest`:

```powershell
npm install --global opencode-ai@latest
opencode --version
$env:OPENCODE_API_KEY = "<secret>"
pwsh -NoProfile -File tests/e2e/opencode-minimal.ps1
```

Set `OPENCODE_API_KEY` as an Actions repository secret to enable the live job. The workflow runs the offline test on a `windows-latest` / `ubuntu-latest` matrix on every `push`, `pull_request`, manual run, and schedule. The live test uses the same matrix, runs only on manual or scheduled runs after the offline test and when the secret is present. It never runs on pull requests.

Action logs are published as diagnostic artifacts. `TestResults/` also holds diagnostics from a local live run and is git-ignored.

The live test does not run on every PR because it consumes a credentialed, billable external API and can introduce transient failures that are not installer regressions.
