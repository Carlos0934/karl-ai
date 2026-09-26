# Karl AI Agent Dotfiles

Orchestrated OpenCode agents with self-contained prompts. `karl-orchestrator` runs as the primary session (`mode: primary`) and routes phase work to managed subagents. Worker, scout, verify, and reviewer run only as subagents and never delegate further:

```text
Select the karl-orchestrator agent as the primary session.
The orchestrator routes mapping to karl-scout, bounded writes to karl-worker,
exact re-execution to karl-verify, and independent evaluation to karl-reviewer.
```

ORCHESTRATOR owns routing, scope, and the final ready/not-ready decision. WORKER owns a change inside the given scope. VERIFY re-runs the exact verification commands and returns evidence without a verdict. REVIEWER evaluates the result against the acceptance criteria without repairing. SCOUT returns cited facts, gaps, and dead ends without recommendations. Each agent prompt is self-contained: role, procedure, safety, and return contract live in the agent file. External non-Karl skills remain allowed.

## Distribution

| Content | OpenCode |
|---|---|
| Entry rules | None (routing lives in the orchestrator agent prompt, no managed block) |
| Orchestrator | `harnesses/opencode/agents/karl-orchestrator.md`, `mode: primary` |
| Worker | `harnesses/opencode/agents/karl-worker.md`, `mode: subagent` |
| Verify | `harnesses/opencode/agents/karl-verify.md`, `mode: subagent` |
| Reviewer | `harnesses/opencode/agents/karl-reviewer.md`, `mode: subagent` |
| Scout | `harnesses/opencode/agents/karl-scout.md`, `mode: subagent` |

Files under `harnesses/opencode/agents/` hold the full agent context: permissions, mode, role, procedure, safety, and return contract. There are no `karl-*` skills; each prompt is self-contained.

Agent configuration (including model selection) lives in `harnesses/` and is not documented here.

## Install

The installer is split by platform: `install.ps1` is Windows-only, `install.sh` covers Linux and macOS. Both installers have the same contract and the same flag semantics.

On Windows, PowerShell 7 is required and Developer Mode must be enabled to create symlinks without elevation. On Linux and macOS, the installer uses POSIX sh and normal symlink support and does not query the Windows registry.

Windows:

```powershell
pwsh -File "$HOME/.agents/install.ps1" -DryRun
pwsh -File "$HOME/.agents/install.ps1"
```

Linux and macOS:

```sh
sh install.sh --dry-run
sh install.sh
```

`install.sh` flags mirror the PowerShell switches: `--dry-run` (`-DryRun`), `--force` (`-Force`), `--uninstall` (`-Uninstall`), `--target-home <dir>` (`-TargetHome`), and `--repo <url-or-path>` (`-RepoUrl`).

### Install from GitHub URL

Both installers can bootstrap the repository themselves — no manual `git clone` step:

Linux and macOS:

```sh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/Carlos0934/karl-ai/master/install.sh)" -- --repo https://github.com/Carlos0934/karl-ai.git
```

Windows (PowerShell 7):

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Carlos0934/karl-ai/master/install.ps1))) -RepoUrl https://github.com/Carlos0934/karl-ai.git
```

`--repo` (`-RepoUrl`) clones the repository into `~/.agents` and runs the normal install from that clone. Private repositories use your existing git credentials, or a token-embedded HTTPS URL (never commit one). Re-running the same command updates the clone with `git pull --ff-only`. If `~/.agents` exists but is not a matching clone, the installer refuses; re-run with `--force` (`-Force`) to back it up and re-clone.

The installer:

1. Validates the agent frontmatter (`mode`, `description`, `permissions`).
2. Removes any legacy block delimited by `karl-ai` comments in global `AGENTS.md` files (OpenCode-only now; Codex support was dropped).
3. Creates per-file symlinks for the OpenCode custom agents.
4. Removes legacy Codex `karl-*.toml` links.
5. Keeps backups under `~/.agents-backup/<timestamp>/` before replacing content.

If a custom agent already exists and is not the expected symlink, the installer stops. Use `-Force` to back it up and replace it.

## Uninstall

Windows:

```powershell
pwsh -File "$HOME/.agents/install.ps1" -Uninstall -DryRun
pwsh -File "$HOME/.agents/install.ps1" -Uninstall
```

Linux and macOS:

```sh
sh install.sh --uninstall --dry-run
sh install.sh --uninstall
```

Uninstall removes only:

- blocks delimited by `<!-- karl-ai: controlled-development -->`
- symlinks whose target is inside this repository

Uninstall does not remove other rules, skills, or agents.

## Sync to Another Machine

```powershell
git clone <private-repository> "$HOME/.agents"
pwsh -File "$HOME/.agents/install.ps1"
```

On Linux or macOS run `sh install.sh` instead of the `pwsh` command after cloning.

External skills installed by other managers stay local and are not part of this repository. This repository ships no skills; agent context lives in the agent files.

## Skill Diagnostics

OpenCode does not load `~/.agents/skills` when started with:

```text
OPENCODE_DISABLE_EXTERNAL_SKILLS=1
```

The installer warns if that variable is detected. Remove it from the environment before starting OpenCode; no need to duplicate or link skills into `~/.config/opencode/skills`.

## E2E

The offline tests run the full install/uninstall cycle in a temporary home. No credentials, Pester, or other dependencies required.

Windows (pwsh):

```powershell
pwsh -NoProfile -File tests/e2e/install-lifecycle.ps1
```

Linux and macOS (sh):

```sh
sh tests/e2e/install-lifecycle.sh
```

From any host you can reproduce the CI Debian install/uninstall run with Docker:

```sh
docker run --rm --mount "type=bind,source=$PWD,target=/repo" -w /repo debian:bookworm-slim sh tests/e2e/install-lifecycle.sh
```

CI validates only the install/uninstall lifecycle in three environments on every `push`, `pull_request`, manual run, and schedule:

- `windows-latest` with pwsh (`install.ps1`, Windows-only)
- `debian:bookworm-slim` container (on an Ubuntu runner) with POSIX sh, plus shellcheck on `install.sh` and the sh E2E
- `macos-latest` with POSIX sh, plus the same shellcheck

Action logs are published as diagnostic artifacts. `TestResults/` is git-ignored.
