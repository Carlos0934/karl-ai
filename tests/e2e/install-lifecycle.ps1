[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path (Join-Path $PSScriptRoot "..") "..")).Path
$Installer = Join-Path $RepoRoot "install.ps1"
$TestHome = Join-Path ([System.IO.Path]::GetTempPath()) "karl-install-e2e-$([guid]::NewGuid().ToString('N'))"
$MarkerStart = "<!-- karl-ai: controlled-development -->"
$MarkerEnd = "<!-- /karl-ai: controlled-development -->"
$ManagedPattern = "(?s)$([regex]::Escape($MarkerStart)).*?$([regex]::Escape($MarkerEnd))"
$PathComparison = if ($IsWindows) { [System.StringComparison]::OrdinalIgnoreCase } else { [System.StringComparison]::Ordinal }

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) { throw $Message }
}

function Join-PathSegments {
    param([string]$Base, [string[]]$Segments)

    $path = $Base
    foreach ($segment in $Segments) {
        $path = Join-Path $path $segment
    }
    return $path
}

function Invoke-TestGit {
    # Runs git with output swallowed; EAP is relaxed around the native call so
    # redirected stderr cannot trip EAP=Stop.
    param([string]$RepoPath, [string[]]$GitArguments)

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        if ($RepoPath) {
            & git -C $RepoPath @GitArguments 2>&1 | Out-Null
        } else {
            & git @GitArguments 2>&1 | Out-Null
        }
        if ($LASTEXITCODE -ne 0) {
            throw "git $($GitArguments -join ' ') failed with exit code $LASTEXITCODE."
        }
    } finally {
        $ErrorActionPreference = $previous
    }
}

$canonicalRules = [System.IO.File]::ReadAllText((Join-Path $RepoRoot "AGENTS.md"))
$canonicalBlockMatch = [regex]::Match($canonicalRules, $ManagedPattern)
Assert-True $canonicalBlockMatch.Success "Canonical AGENTS.md does not contain the managed block."
$CanonicalManagedBlock = $canonicalBlockMatch.Value
$ExpectedLinks = @(
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".config", "opencode", "agents", "karl-orchestrator.md"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "opencode", "agents", "karl-orchestrator.md") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".config", "opencode", "agents", "karl-worker.md"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "opencode", "agents", "karl-worker.md") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".config", "opencode", "agents", "karl-reviewer.md"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "opencode", "agents", "karl-reviewer.md") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".config", "opencode", "agents", "karl-scout.md"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "opencode", "agents", "karl-scout.md") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".codex", "agents", "karl-worker.toml"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "codex", "agents", "karl-worker.toml") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".codex", "agents", "karl-reviewer.toml"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "codex", "agents", "karl-reviewer.toml") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".codex", "agents", "karl-scout.toml"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "codex", "agents", "karl-scout.toml") }
)

function Get-LinkTarget {
    param([string]$Path)
    $item = Get-Item -LiteralPath $Path -Force
    Assert-True ($item.LinkType -eq "SymbolicLink") "Expected a symbolic link at '$Path'."
    $target = @($item.Target)[0]
    if (-not [System.IO.Path]::IsPathRooted($target)) {
        $target = Join-Path $item.DirectoryName $target
    }
    [System.IO.Path]::GetFullPath($target)
}

function Assert-InstalledState {
    foreach ($agentsPath in @(
        Join-PathSegments $TestHome @(".config", "opencode", "AGENTS.md")
        Join-PathSegments $TestHome @(".codex", "AGENTS.md")
    )) {
        $content = [System.IO.File]::ReadAllText($agentsPath)
        $installedBlocks = [regex]::Matches($content, $ManagedPattern)
        Assert-True ($installedBlocks.Count -eq 1) "Expected exactly one managed block in '$agentsPath', found $($installedBlocks.Count)."
        $targetNewline = if ($content.Contains("`r`n")) { "`r`n" } else { "`n" }
        $expectedBlock = $CanonicalManagedBlock -replace "`r?`n", $targetNewline
        Assert-True ($installedBlocks[0].Value -ceq $expectedBlock) "Managed block in '$agentsPath' does not exactly match canonical AGENTS.md content."
    }

    Assert-True ($ExpectedLinks.Count -eq 7) "The test must cover exactly seven managed links."
    $actualLinks = @(Get-ChildItem -LiteralPath (Join-PathSegments $TestHome @(".config", "opencode", "agents")), (Join-PathSegments $TestHome @(".codex", "agents")) -File -Force | Where-Object { $_.LinkType -eq "SymbolicLink" })
    Assert-True ($actualLinks.Count -eq 7) "Expected exactly seven managed links, found $($actualLinks.Count)."
    foreach ($entry in $ExpectedLinks.GetEnumerator()) {
        $actual = Get-LinkTarget $entry.TargetPath
        $expected = [System.IO.Path]::GetFullPath($entry.SourcePath)
        Assert-True ($actual.Equals($expected, $PathComparison)) "Wrong link target for '$($entry.TargetPath)': '$actual'."
    }
}

function Get-ManagedFingerprint {
    @(Get-ChildItem -LiteralPath $TestHome -Recurse -Force | ForEach-Object {
        $relative = $_.FullName.Substring($TestHome.Length)
        if ($_.LinkType -eq "SymbolicLink") {
            "L:${relative}:$(Get-LinkTarget $_.FullName)"
        } elseif ($_.PSIsContainer) {
            "D:$relative"
        } else {
            "F:${relative}:$((Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash)"
        }
    } | Sort-Object) -join "`n"
}

try {
    $openCodeRoot = Join-PathSegments $TestHome @(".config", "opencode")
    $codexRoot = Join-Path $TestHome ".codex"
    $unrelatedSkill = Join-PathSegments $TestHome @(".agents", "skills", "unrelated")
    New-Item -ItemType Directory -Path $openCodeRoot, $codexRoot, $unrelatedSkill -Force | Out-Null

    $openCodeOriginal = "# Existing OpenCode rules`n`nKeep OpenCode content.`n"
    $codexOriginal = "# Existing Codex rules`n`nKeep Codex content.`n"
    $skillOriginal = "---`nname: unrelated`n---`n`nUnrelated skill content.`n"
    [System.IO.File]::WriteAllText((Join-Path $openCodeRoot "AGENTS.md"), $openCodeOriginal)
    [System.IO.File]::WriteAllText((Join-Path $codexRoot "AGENTS.md"), $codexOriginal)
    [System.IO.File]::WriteAllText((Join-Path $unrelatedSkill "SKILL.md"), $skillOriginal)

    & $Installer -TargetHome $TestHome -SkipDeveloperModeCheck
    Assert-InstalledState
    Assert-True (Test-Path -LiteralPath (Join-Path $TestHome ".agents-backup") -PathType Container) "Backups were not rooted under TargetHome."
    $installed = Get-ManagedFingerprint

    & $Installer -TargetHome $TestHome -SkipDeveloperModeCheck
    Assert-InstalledState
    Assert-True ((Get-ManagedFingerprint) -ceq $installed) "A repeated install changed managed state or created another backup."

    & $Installer -TargetHome $TestHome -SkipDeveloperModeCheck -Uninstall -DryRun
    Assert-True ((Get-ManagedFingerprint) -ceq $installed) "Uninstall dry-run mutated the test home."

    & $Installer -TargetHome $TestHome -SkipDeveloperModeCheck -Uninstall
    foreach ($entry in $ExpectedLinks) {
        Assert-True (-not (Test-Path -LiteralPath $entry.TargetPath)) "Managed link '$($entry.TargetPath)' survived uninstall."
    }
    $openCodeAfter = [System.IO.File]::ReadAllText((Join-Path $openCodeRoot "AGENTS.md"))
    $codexAfter = [System.IO.File]::ReadAllText((Join-Path $codexRoot "AGENTS.md"))
    Assert-True (-not $openCodeAfter.Contains($MarkerStart)) "OpenCode managed block survived uninstall."
    Assert-True (-not $codexAfter.Contains($MarkerStart)) "Codex managed block survived uninstall."
    Assert-True ($openCodeAfter -ceq $openCodeOriginal) "Existing OpenCode content was not preserved exactly."
    Assert-True ($codexAfter -ceq $codexOriginal) "Existing Codex content was not preserved exactly."
    Assert-True ([System.IO.File]::ReadAllText((Join-Path $unrelatedSkill "SKILL.md")) -ceq $skillOriginal) "The unrelated skill changed."

    # Regression: Assert-KarlSkills must accept CRLF skill files (GitHub Windows runner default).
    # This runs the REAL install.ps1 code path against a temp repo whose SKILL.md files are CRLF.
    $crlfRepoRoot = Join-Path $TestHome "crlf-repo"
    $crlfTargetHome = Join-Path $TestHome "crlf-target"
    $crlfOpenCodeRoot = Join-PathSegments $crlfTargetHome @(".config", "opencode")
    $crlfCodexRoot = Join-Path $crlfTargetHome ".codex"

    foreach ($segment in @("AGENTS.md", "install.ps1", "skills", "harnesses")) {
        $source = Join-Path $RepoRoot $segment
        $destination = Join-Path $crlfRepoRoot $segment
        if ((Get-Item -LiteralPath $source -Force).PSIsContainer) {
            Copy-Item -LiteralPath $source -Destination $destination -Recurse -Force
        } else {
            New-Item -ItemType Directory -Path (Split-Path -Parent $destination) -Force | Out-Null
            Copy-Item -LiteralPath $source -Destination $destination -Force
        }
    }

    foreach ($skillFile in (Get-ChildItem -LiteralPath (Join-Path $crlfRepoRoot "skills") -Recurse -Filter "SKILL.md" -File -Force)) {
        $content = [System.IO.File]::ReadAllText($skillFile.FullName)
        $crlfContent = $content -replace "`r?`n", "`r`n"
        [System.IO.File]::WriteAllText($skillFile.FullName, $crlfContent)
        Assert-True ($crlfContent.Contains("`r`n")) "Failed to convert '$($skillFile.FullName)' to CRLF."
    }

    New-Item -ItemType Directory -Path $crlfOpenCodeRoot, $crlfCodexRoot -Force | Out-Null
    [System.IO.File]::WriteAllText((Join-Path $crlfOpenCodeRoot "AGENTS.md"), "# OpenCode`n`n")
    [System.IO.File]::WriteAllText((Join-Path $crlfCodexRoot "AGENTS.md"), "# Codex`n`n")

    $copiedInstaller = Join-Path $crlfRepoRoot "install.ps1"
    & $copiedInstaller -TargetHome $crlfTargetHome -SkipDeveloperModeCheck
    Write-Host "PASS: Assert-KarlSkills accepts CRLF skill files."

    # --- -RepoUrl bootstrap clone --------------------------------------------
    # Fixture repository: a minimal copy of the real repo (AGENTS.md with the
    # managed block, skills/, harnesses/) committed to a local git repo. The
    # installer is run from the REAL repo but installs from this fixture clone.
    $fixtureRepo = Join-Path $TestHome "fixture-repo"
    foreach ($segment in @("AGENTS.md", "skills", "harnesses")) {
        $source = Join-Path $RepoRoot $segment
        $destination = Join-Path $fixtureRepo $segment
        if ((Get-Item -LiteralPath $source -Force).PSIsContainer) {
            Copy-Item -LiteralPath $source -Destination $destination -Recurse -Force
        } else {
            New-Item -ItemType Directory -Path (Split-Path -Parent $destination) -Force | Out-Null
            Copy-Item -LiteralPath $source -Destination $destination -Force
        }
    }
    Invoke-TestGit -RepoPath $fixtureRepo -GitArguments @("init")
    Invoke-TestGit -RepoPath $fixtureRepo -GitArguments @("config", "user.name", "Karl E2E")
    Invoke-TestGit -RepoPath $fixtureRepo -GitArguments @("config", "user.email", "karl-e2e@example.invalid")
    Invoke-TestGit -RepoPath $fixtureRepo -GitArguments @("add", "-A")
    Invoke-TestGit -RepoPath $fixtureRepo -GitArguments @("commit", "-m", "fixture")

    $repoTargetHome = Join-Path $TestHome "repo-target"
    $repoOpenCodeRoot = Join-PathSegments $repoTargetHome @(".config", "opencode")
    $repoCodexRoot = Join-Path $repoTargetHome ".codex"
    New-Item -ItemType Directory -Path $repoOpenCodeRoot, $repoCodexRoot -Force | Out-Null
    [System.IO.File]::WriteAllText((Join-Path $repoOpenCodeRoot "AGENTS.md"), "# OpenCode`n`n")
    [System.IO.File]::WriteAllText((Join-Path $repoCodexRoot "AGENTS.md"), "# Codex`n`n")

    $cloneDir = Join-Path $repoTargetHome ".agents"
    & $Installer -RepoUrl $fixtureRepo -TargetHome $repoTargetHome -SkipDeveloperModeCheck

    Assert-True (Test-Path -LiteralPath (Join-Path $cloneDir "AGENTS.md")) "-RepoUrl did not clone the fixture into '$cloneDir'."
    Assert-True (Test-Path -LiteralPath (Join-PathSegments $cloneDir @("skills", "karl-orchestrate", "SKILL.md"))) "The clone at '$cloneDir' is missing the Karl skills."
    foreach ($blockPath in @((Join-Path $repoOpenCodeRoot "AGENTS.md"), (Join-Path $repoCodexRoot "AGENTS.md"))) {
        $content = [System.IO.File]::ReadAllText($blockPath)
        $blocks = [regex]::Matches($content, $ManagedPattern)
        Assert-True ($blocks.Count -eq 1) "Expected exactly one managed block in '$blockPath', found $($blocks.Count)."
        Assert-True ($blocks[0].Value -ceq $CanonicalManagedBlock) "Managed block in '$blockPath' does not match canonical AGENTS.md content."
    }
    foreach ($name in @("karl-orchestrator.md", "karl-worker.md", "karl-reviewer.md", "karl-scout.md")) {
        $linkPath = Join-Path $repoOpenCodeRoot "agents/$name"
        $actual = Get-LinkTarget $linkPath
        $expected = [System.IO.Path]::GetFullPath((Join-PathSegments $cloneDir @("harnesses", "opencode", "agents", $name)))
        Assert-True ($actual.Equals($expected, $PathComparison)) "Wrong link target for '$linkPath': '$actual' (expected a link into the clone dir)."
    }
    Write-Host "PASS: -RepoUrl clones the fixture into the target home and installs from the clone."

    # Second run with the same switch: pull (no-op) plus already-current state
    # and no new backup.
    $repoBackupDir = Join-Path $repoTargetHome ".agents-backup"
    $backupsBefore = @(Get-ChildItem -LiteralPath $repoBackupDir -Directory -Force -ErrorAction SilentlyContinue).Count
    & $Installer -RepoUrl $fixtureRepo -TargetHome $repoTargetHome -SkipDeveloperModeCheck
    $backupsAfter = @(Get-ChildItem -LiteralPath $repoBackupDir -Directory -Force -ErrorAction SilentlyContinue).Count
    Assert-True ($backupsBefore -eq $backupsAfter) "A repeated -RepoUrl install created another backup."
    foreach ($blockPath in @((Join-Path $repoOpenCodeRoot "AGENTS.md"), (Join-Path $repoCodexRoot "AGENTS.md"))) {
        $blocks = [regex]::Matches([System.IO.File]::ReadAllText($blockPath), $ManagedPattern)
        Assert-True ($blocks.Count -eq 1) "Expected exactly one managed block in '$blockPath' after the repeated -RepoUrl install."
    }
    Write-Host "PASS: a repeated -RepoUrl install updates the clone and is a no-op."

    # Mismatch refusal: .agents is a git repo but not a clone of the fixture URL.
    $localOnly = Join-Path $cloneDir "local-only.txt"
    [System.IO.File]::WriteAllText($localOnly, "keep me`n")
    Invoke-TestGit -RepoPath $cloneDir -GitArguments @("remote", "remove", "origin")
    $threw = $false
    try { & $Installer -RepoUrl $fixtureRepo -TargetHome $repoTargetHome -SkipDeveloperModeCheck } catch { $threw = $true }
    Assert-True $threw "-RepoUrl install did not refuse a non-matching git repo at '$cloneDir'."
    Assert-True (Test-Path -LiteralPath $localOnly) "The refused -RepoUrl install modified '$cloneDir'."

    # -Force backs the mismatched dir up and re-clones.
    & $Installer -RepoUrl $fixtureRepo -TargetHome $repoTargetHome -SkipDeveloperModeCheck -Force
    Assert-True (Test-Path -LiteralPath (Join-Path $cloneDir "AGENTS.md")) "-Force did not re-clone the fixture into '$cloneDir'."
    Assert-True (-not (Test-Path -LiteralPath $localOnly)) "-Force re-clone kept the mismatching content at '$localOnly'."
    $forcedBackups = @(Get-ChildItem -LiteralPath $repoBackupDir -Recurse -Filter "local-only.txt" -File -Force -ErrorAction SilentlyContinue)
    Assert-True ($forcedBackups.Count -ge 1) "-Force did not back up the mismatched clone dir."
    Assert-True ([System.IO.File]::ReadAllText($forcedBackups[0].FullName) -ceq "keep me`n") "The -Force backup of the mismatched clone dir lost its content."
    Write-Host "PASS: -RepoUrl refuses a non-matching git repo without -Force and re-clones with -Force."

    # Bootstrap requirement: an installer copy outside a checkout must demand
    # -RepoUrl.
    $bareDir = Join-Path $TestHome "bare"
    New-Item -ItemType Directory -Path $bareDir -Force | Out-Null
    Copy-Item -LiteralPath $Installer -Destination (Join-Path $bareDir "install.ps1") -Force
    $threw = $false
    try { & (Join-Path $bareDir "install.ps1") -TargetHome $repoTargetHome -SkipDeveloperModeCheck } catch { $threw = $true }
    Assert-True $threw "-RepoUrl was not required when the installer runs outside a checkout."
    Write-Host "PASS: install.ps1 outside a checkout requires -RepoUrl."

    # Bootstrap requirement: a copied installer beside a bare AGENTS.md (no
    # skills/ or harnesses/ directories) is NOT a checkout: it must also
    # demand -RepoUrl (not run in place from the stray file).
    $semiBareDir = Join-Path $TestHome "semi-bare"
    New-Item -ItemType Directory -Path $semiBareDir -Force | Out-Null
    Copy-Item -LiteralPath $Installer -Destination (Join-Path $semiBareDir "install.ps1") -Force
    [System.IO.File]::WriteAllText((Join-Path $semiBareDir "AGENTS.md"), "# Stray AGENTS.md, not a checkout.`n")
    $semiBareError = $null
    try { & (Join-Path $semiBareDir "install.ps1") -TargetHome $repoTargetHome -SkipDeveloperModeCheck } catch { $semiBareError = $_ }
    Assert-True ($null -ne $semiBareError -and $semiBareError.Exception.Message -like "*-RepoUrl*") `
        "An installer copy beside a bare AGENTS.md did not demand -RepoUrl (got: '$($semiBareError.Exception.Message)')."
    Write-Host "PASS: an installer copy beside a bare AGENTS.md requires -RepoUrl."

    # --- Scriptblock invocation reproducing the README one-liner shape --------
    # & ([scriptblock]::Create((irm <url>))) -RepoUrl <url> has no
    # $PSScriptRoot; the ENTIRE flow (clone, managed block, harness links) must
    # run with the repo root bound to the clone directory. Local fixture stands
    # in for the raw download.
    $scriptblockTargetHome = Join-Path $TestHome "scriptblock-target"
    $scriptblockOpenCodeRoot = Join-PathSegments $scriptblockTargetHome @(".config", "opencode")
    $scriptblockCodexRoot = Join-Path $scriptblockTargetHome ".codex"
    New-Item -ItemType Directory -Path $scriptblockOpenCodeRoot, $scriptblockCodexRoot -Force | Out-Null
    [System.IO.File]::WriteAllText((Join-Path $scriptblockOpenCodeRoot "AGENTS.md"), "# OpenCode`n`n")
    [System.IO.File]::WriteAllText((Join-Path $scriptblockCodexRoot "AGENTS.md"), "# Codex`n`n")

    $bootstrapScriptBlock = [scriptblock]::Create((Get-Content -Raw $Installer))
    & $bootstrapScriptBlock -RepoUrl $fixtureRepo -TargetHome $scriptblockTargetHome -SkipDeveloperModeCheck

    $scriptblockCloneDir = Join-Path $scriptblockTargetHome ".agents"
    Assert-True (Test-Path -LiteralPath (Join-Path $scriptblockCloneDir "AGENTS.md")) `
        "The scriptblock invocation did not clone the fixture into '$scriptblockCloneDir'."
    Assert-True (Test-Path -LiteralPath (Join-PathSegments $scriptblockCloneDir @("skills", "karl-orchestrate", "SKILL.md"))) `
        "The scriptblock invocation clone at '$scriptblockCloneDir' is missing the Karl skills."
    foreach ($blockPath in @((Join-Path $scriptblockOpenCodeRoot "AGENTS.md"), (Join-Path $scriptblockCodexRoot "AGENTS.md"))) {
        $blocks = [regex]::Matches([System.IO.File]::ReadAllText($blockPath), $ManagedPattern)
        Assert-True ($blocks.Count -eq 1) "Expected exactly one managed block in '$blockPath' (scriptblock invocation), found $($blocks.Count)."
        Assert-True ($blocks[0].Value -ceq $CanonicalManagedBlock) "Managed block in '$blockPath' does not match canonical AGENTS.md content (scriptblock invocation)."
    }
    foreach ($name in @("karl-orchestrator.md", "karl-worker.md", "karl-reviewer.md", "karl-scout.md")) {
        $linkPath = Join-Path $scriptblockOpenCodeRoot "agents/$name"
        $actual = Get-LinkTarget $linkPath
        $expected = [System.IO.Path]::GetFullPath((Join-PathSegments $scriptblockCloneDir @("harnesses", "opencode", "agents", $name)))
        Assert-True ($actual.Equals($expected, $PathComparison)) "Wrong link target for '$linkPath': '$actual' (expected a link into the scriptblock-invocation clone dir)."
    }
    Write-Host "PASS: scriptblock invocation (README one-liner shape) clones the fixture and installs from the clone."

    Write-Host "PASS: install lifecycle is isolated, idempotent, and non-destructive."
} finally {
    if (Test-Path -LiteralPath $TestHome) {
        Remove-Item -LiteralPath $TestHome -Recurse -Force
    }
}
