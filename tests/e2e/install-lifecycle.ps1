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

$canonicalRules = [System.IO.File]::ReadAllText((Join-Path $RepoRoot "AGENTS.md"))
$canonicalBlockMatch = [regex]::Match($canonicalRules, $ManagedPattern)
Assert-True $canonicalBlockMatch.Success "Canonical AGENTS.md does not contain the managed block."
$CanonicalManagedBlock = $canonicalBlockMatch.Value
$ExpectedLinks = @(
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".config", "opencode", "agents", "karl-orchestrator.md"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "opencode", "agents", "karl-orchestrator.md") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".config", "opencode", "agents", "karl-worker.md"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "opencode", "agents", "karl-worker.md") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".config", "opencode", "agents", "karl-reviewer.md"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "opencode", "agents", "karl-reviewer.md") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".codex", "agents", "karl-worker.toml"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "codex", "agents", "karl-worker.toml") }
    [pscustomobject]@{ TargetPath = Join-PathSegments $TestHome @(".codex", "agents", "karl-reviewer.toml"); SourcePath = Join-PathSegments $RepoRoot @("harnesses", "codex", "agents", "karl-reviewer.toml") }
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

    Assert-True ($ExpectedLinks.Count -eq 5) "The test must cover exactly five managed links."
    $actualLinks = @(Get-ChildItem -LiteralPath (Join-PathSegments $TestHome @(".config", "opencode", "agents")), (Join-PathSegments $TestHome @(".codex", "agents")) -File -Force | Where-Object { $_.LinkType -eq "SymbolicLink" })
    Assert-True ($actualLinks.Count -eq 5) "Expected exactly five managed links, found $($actualLinks.Count)."
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

    Write-Host "PASS: install lifecycle is isolated, idempotent, and non-destructive."
} finally {
    if (Test-Path -LiteralPath $TestHome) {
        Remove-Item -LiteralPath $TestHome -Recurse -Force
    }
}
