# Karl AI Agent Dotfiles

Global, portable configuration for a controlled development topology:

```text
USER -> ORCHESTRATOR -> WORKER -> ORCHESTRATOR -> REVIEWER -> ORCHESTRATOR
```

Only ORCHESTRATOR coordinates. WORKER owns the change. REVIEWER evaluates the result independently and does not repair. Repair cycles are bounded at two.

## The Main Agent: ORCHESTRATOR

The main agent is called **ORCHESTRATOR**. In older material the same role appears as `MAIN`; that name is retired. One role, one name: the entry point the user talks to, and the only agent that coordinates work.

The ORCHESTRATOR does not exist to do the work itself. Its function is:

```text
understand -> delegate -> collect -> decide -> route
```

It owns:

- user intent and the original request
- the expected outcome and acceptance criteria
- workflow state, stage transitions, and the repair count
- delegation to subagents and routing between them
- the final response and the terminal decision

It routes WORKER outcomes to REVIEWER for independent evaluation, and routes REVIEWER findings back to WORKER as repair work. Subagents never talk to each other; every channel passes through the ORCHESTRATOR.

It decides one of three terminal states:

- `COMPLETED` — outcome implemented, REVIEWER returned PASS, nothing blocking
- `BLOCKED` — work cannot continue safely without information, authority, or an external decision
- `UNRESOLVED` — two repair attempts did not reach PASS; the system stops instead of looping

The one exception to delegation: the ORCHESTRATOR may handle trivial, non-behavioral work directly when independent review would add no value (a typo, pure formatting, a mechanical edit).

WORKER and REVIEWER are subagents, not co-workers: WORKER may change the system inside the delegated scope, REVIEWER may evaluate it and return `PASS` / `FAIL` / `BLOCKED` with evidence, and REVIEWER never repairs. The full procedures live in the skills (`karl-orchestrate`, `karl-work`, `karl-review`).

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

1. Validates `karl-*` skills.
2. Inserts or updates only the block delimited by `karl-ai` comments in each global `AGENTS.md`.
3. Creates per-file symlinks for each harness's custom agents.
4. Keeps backups under `~/.agents-backup/<timestamp>/` before replacing content.

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
- `karl-*` skill directories under `~/.agents/skills/` (only directories containing a `SKILL.md`; unrelated skills survive)

Warning: because uninstall removes the `karl-*` skill directories, on the primary machine those are git-tracked repository files. Restore them with `git checkout` (for example `git checkout -- skills/`) after an uninstall. Uninstall does not remove other rules, skills, or agents.

## Sync to Another Machine

```powershell
git clone <private-repository> "$HOME/.agents"
pwsh -File "$HOME/.agents/install.ps1"
```

On Linux or macOS run `sh install.sh` instead of the `pwsh` command after cloning.

Installs or updates skills directly inside `~/.agents/skills`; both harnesses discover them with no extra steps.

External skills installed by other managers stay local and are not part of this repository. Git only versions the Karl skills maintained here.

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

From any host you can reproduce the CI Debian run with Docker:

```sh
docker run --rm --mount "type=bind,source=$PWD,target=/repo" -w /repo debian:bookworm-slim sh tests/e2e/install-lifecycle.sh
```

The live test requires `OPENCODE_API_KEY`. It uses a temporary home and a temporary git repository, runs `karl-orchestrator` non-interactively, and keeps JSON stdout and stderr under `TestResults/`. Policy is to install `opencode-ai@latest`:

```powershell
npm install --global opencode-ai@latest
opencode --version
$env:OPENCODE_API_KEY = "<secret>"
pwsh -NoProfile -File tests/e2e/opencode-minimal.ps1
```

Set `OPENCODE_API_KEY` as an Actions repository secret to enable the live job. CI runs the offline tests in three environments on every `push`, `pull_request`, manual run, and schedule:

- `windows-latest` with pwsh (`install.ps1`, Windows-only)
- `debian:bookworm-slim` container (on an Ubuntu runner) with POSIX sh, plus shellcheck on `install.sh` and the sh E2E
- `macos-latest` with POSIX sh, plus the same shellcheck

The live test uses a `windows-latest` / `ubuntu-latest` matrix, runs only on manual or scheduled runs after the offline jobs pass and when the secret is present. It never runs on pull requests.

Action logs are published as diagnostic artifacts. `TestResults/` also holds diagnostics from a local live run and is git-ignored.

The live test does not run on every PR because it consumes a credentialed, billable external API and can introduce transient failures that are not installer regressions.
