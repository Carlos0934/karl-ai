[CmdletBinding()]
param(
    [switch]$DryRun,
    [switch]$Force,
    [switch]$Uninstall,
    [string]$TargetHome = $HOME,
    [switch]$SkipDeveloperModeCheck
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = $PSScriptRoot
$TargetHome = [System.IO.Path]::GetFullPath($TargetHome)
$MarkerStart = "<!-- karl-ai: controlled-development -->"
$MarkerEnd = "<!-- /karl-ai: controlled-development -->"
$ManagedPattern = "(?s)$([regex]::Escape($MarkerStart)).*?$([regex]::Escape($MarkerEnd))"
$BackupRoot = Join-Path (Join-Path $TargetHome ".agents-backup") (Get-Date -Format 'yyyyMMdd-HHmmss')
$PathComparison = if ($IsWindows) { [System.StringComparison]::OrdinalIgnoreCase } else { [System.StringComparison]::Ordinal }
$DirectorySeparators = [char[]]@([System.IO.Path]::DirectorySeparatorChar, [System.IO.Path]::AltDirectorySeparatorChar)

function Write-Action {
    param([string]$Message)

    $prefix = if ($DryRun) { "[DRY RUN]" } else { "[OK]" }
    Write-Host "$prefix $Message"
}

function Invoke-Mutation {
    param(
        [string]$Description,
        [scriptblock]$Action
    )

    if (-not $DryRun) {
        & $Action
    }
    Write-Action $Description
}

function Ensure-Directory {
    param([string]$Path)

    $item = Get-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue
    if ($item -and -not $item.PSIsContainer) {
        throw "Expected a directory at '$Path'."
    }
    if (-not $item) {
        Invoke-Mutation "Create directory $Path" {
            New-Item -ItemType Directory -Path $Path -Force | Out-Null
        }
    }
}

function Backup-File {
    param(
        [string]$Path,
        [string]$RelativeBackupPath
    )

    if (-not (Test-Path -LiteralPath $Path)) {
        return
    }

    $backupPath = Join-Path $BackupRoot $RelativeBackupPath
    $backupParent = Split-Path -Parent $backupPath
    New-Item -ItemType Directory -Path $backupParent -Force | Out-Null
    Copy-Item -LiteralPath $Path -Destination $backupPath -Force
}

function Get-NormalizedLinkTarget {
    param([System.IO.FileSystemInfo]$Item)

    $target = @($Item.Target)[0]
    if (-not $target) {
        return $null
    }
    if (-not [System.IO.Path]::IsPathRooted($target)) {
        $target = Join-Path $Item.DirectoryName $target
    }
    return [System.IO.Path]::GetFullPath($target)
}

function Update-ManagedBlock {
    param(
        [string]$TargetPath,
        [string]$BackupPath,
        [string]$ManagedBlock
    )

    $existing = if (Test-Path -LiteralPath $TargetPath) {
        [System.IO.File]::ReadAllText($TargetPath)
    } else {
        ""
    }
    $newline = if ($existing.Contains("`r`n")) { "`r`n" } else { "`n" }
    $block = $ManagedBlock -replace "`r?`n", $newline

    if ($Uninstall) {
        if (-not [regex]::IsMatch($existing, $ManagedPattern)) {
            Write-Action "Managed block already absent from $TargetPath"
            return
        }

        $updated = [regex]::Replace($existing, $ManagedPattern, "").TrimEnd() + $newline
        Invoke-Mutation "Remove managed block from $TargetPath" {
            Backup-File $TargetPath $BackupPath
            if ([string]::IsNullOrWhiteSpace($updated)) {
                Remove-Item -LiteralPath $TargetPath -Force
            } else {
                [System.IO.File]::WriteAllText($TargetPath, $updated, [System.Text.UTF8Encoding]::new($false))
            }
        }
        return
    }

    $updated = if ([regex]::IsMatch($existing, $ManagedPattern)) {
        [regex]::Replace($existing, $ManagedPattern, [System.Text.RegularExpressions.MatchEvaluator]{ param($match) $block })
    } elseif ([string]::IsNullOrWhiteSpace($existing)) {
        $block + $newline
    } else {
        $existing.TrimEnd() + $newline + $newline + $block + $newline
    }

    if ($updated -ceq $existing) {
        Write-Action "Managed block already current in $TargetPath"
        return
    }

    Invoke-Mutation "Install managed block in $TargetPath" {
        Backup-File $TargetPath $BackupPath
        $parent = Split-Path -Parent $TargetPath
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
        [System.IO.File]::WriteAllText($TargetPath, $updated, [System.Text.UTF8Encoding]::new($false))
    }
}

function Install-AgentLink {
    param(
        [string]$SourcePath,
        [string]$TargetPath,
        [string]$BackupPath
    )

    $normalizedSource = [System.IO.Path]::GetFullPath($SourcePath)
    $existing = Get-Item -LiteralPath $TargetPath -Force -ErrorAction SilentlyContinue

    if ($existing -and $existing.LinkType -eq "SymbolicLink") {
        $currentTarget = Get-NormalizedLinkTarget $existing
        if ($currentTarget -and $currentTarget.Equals($normalizedSource, $PathComparison)) {
            Write-Action "Link already current: $TargetPath"
            return
        }
    }

    if ($existing -and -not $Force) {
        throw "Destination '$TargetPath' already exists. Re-run with -Force to back it up and replace it."
    }

    Invoke-Mutation "Link $TargetPath -> $SourcePath" {
        if ($existing) {
            Backup-File $TargetPath $BackupPath
            Remove-Item -LiteralPath $TargetPath -Force
        }
        New-Item -ItemType SymbolicLink -Path $TargetPath -Target $SourcePath | Out-Null
    }
}

function Remove-OwnedLinks {
    param(
        [string]$SourceDirectory,
        [string]$TargetDirectory
    )

    if (-not (Test-Path -LiteralPath $TargetDirectory)) {
        return
    }

    $normalizedSourceDirectory = [System.IO.Path]::GetFullPath($SourceDirectory).TrimEnd($DirectorySeparators) + [System.IO.Path]::DirectorySeparatorChar
    foreach ($item in Get-ChildItem -LiteralPath $TargetDirectory -File -Force) {
        if ($item.LinkType -ne "SymbolicLink") {
            continue
        }
        $target = Get-NormalizedLinkTarget $item
        if ($target -and $target.StartsWith($normalizedSourceDirectory, $PathComparison)) {
            Invoke-Mutation "Remove managed link $($item.FullName)" {
                Remove-Item -LiteralPath $item.FullName -Force
            }
        }
    }
}

function Sync-HarnessAgents {
    param(
        [string]$Harness,
        [string]$HarnessRoot,
        [string]$SourceDirectory,
        [string]$Pattern
    )

    if (-not (Test-Path -LiteralPath $HarnessRoot)) {
        Write-Warning "Skipping $Harness because '$HarnessRoot' does not exist."
        return
    }

    $targetDirectory = Join-Path $HarnessRoot "agents"
    if ($Uninstall) {
        Remove-OwnedLinks $SourceDirectory $targetDirectory
        return
    }

    Ensure-Directory $targetDirectory
    $sourceFiles = @(Get-ChildItem -LiteralPath $SourceDirectory -File -Filter $Pattern)
    $sourceNames = [System.Collections.Generic.HashSet[string]]::new(
        $(if ($IsWindows) { [System.StringComparer]::OrdinalIgnoreCase } else { [System.StringComparer]::Ordinal })
    )
    foreach ($source in $sourceFiles) {
        [void]$sourceNames.Add($source.Name)
    }

    $normalizedSourceDirectory = [System.IO.Path]::GetFullPath($SourceDirectory).TrimEnd($DirectorySeparators) + [System.IO.Path]::DirectorySeparatorChar
    foreach ($item in Get-ChildItem -LiteralPath $targetDirectory -File -Force) {
        if ($item.LinkType -ne "SymbolicLink") {
            continue
        }
        $linkTarget = Get-NormalizedLinkTarget $item
        if (-not $linkTarget -or -not $linkTarget.StartsWith($normalizedSourceDirectory, $PathComparison)) {
            continue
        }
        if (-not $sourceNames.Contains($item.Name)) {
            Invoke-Mutation "Remove stale managed link $($item.FullName)" {
                Remove-Item -LiteralPath $item.FullName -Force
            }
        }
    }

    foreach ($source in $sourceFiles) {
        $target = Join-Path $targetDirectory $source.Name
        Install-AgentLink $source.FullName $target (Join-Path (Join-Path $Harness "agents") $source.Name)
    }
}

function Remove-KarlSkills {
    $skillRoot = Join-Path (Join-Path $TargetHome ".agents") "skills"
    if (-not (Test-Path -LiteralPath $skillRoot)) {
        return
    }
    foreach ($directory in Get-ChildItem -LiteralPath $skillRoot -Directory -Filter "karl-*") {
        $skillFile = Join-Path $directory.FullName "SKILL.md"
        if (-not (Test-Path -LiteralPath $skillFile)) {
            continue
        }
        Invoke-Mutation "Remove skill $($directory.FullName)" {
            Remove-Item -LiteralPath $directory.FullName -Recurse -Force
        }
    }
}

function Assert-KarlSkills {
    $skillRoot = Join-Path $RepoRoot "skills"
    foreach ($directory in Get-ChildItem -LiteralPath $skillRoot -Directory -Filter "karl-*") {
        $skillFile = Join-Path $directory.FullName "SKILL.md"
        if (-not (Test-Path -LiteralPath $skillFile)) {
            throw "Missing SKILL.md in '$($directory.FullName)'."
        }
        $content = [System.IO.File]::ReadAllText($skillFile)
        $nameMatch = [regex]::Match($content, "(?m)^name:\s*([^\r\n]+)\r?$")
        if (-not $nameMatch.Success -or $nameMatch.Groups[1].Value.Trim() -cne $directory.Name) {
            throw "Skill name in '$skillFile' must match directory '$($directory.Name)'."
        }
    }
    Write-Action "Validated Karl skills"
}

if (-not $Uninstall) {
    if ($IsWindows -and -not $SkipDeveloperModeCheck) {
        $developerMode = (Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock" -Name "AllowDevelopmentWithoutDevLicense" -ErrorAction SilentlyContinue).AllowDevelopmentWithoutDevLicense
        if ($developerMode -ne 1) {
            throw "Windows Developer Mode is required for unprivileged symbolic links."
        }
    }
    if ($env:OPENCODE_DISABLE_EXTERNAL_SKILLS -eq "1") {
        Write-Warning "OPENCODE_DISABLE_EXTERNAL_SKILLS=1 prevents OpenCode from discovering ~/.agents/skills. Remove that environment variable before starting OpenCode."
    }
}

$managedBlock = ""
if (-not $Uninstall) {
    $canonicalRules = [System.IO.File]::ReadAllText((Join-Path $RepoRoot "AGENTS.md"))
    $blockMatch = [regex]::Match($canonicalRules, $ManagedPattern)
    if (-not $blockMatch.Success) {
        throw "Canonical AGENTS.md does not contain the managed karl-ai block."
    }
    $managedBlock = $blockMatch.Value
    Assert-KarlSkills
}

$openCodeRoot = Join-Path (Join-Path $TargetHome ".config") "opencode"
$codexRoot = Join-Path $TargetHome ".codex"

if (Test-Path -LiteralPath $openCodeRoot) {
    Update-ManagedBlock (Join-Path $openCodeRoot "AGENTS.md") (Join-Path "opencode" "AGENTS.md") $managedBlock
}
if (Test-Path -LiteralPath $codexRoot) {
    Update-ManagedBlock (Join-Path $codexRoot "AGENTS.md") (Join-Path "codex" "AGENTS.md") $managedBlock
}

Sync-HarnessAgents "opencode" $openCodeRoot (Join-Path (Join-Path (Join-Path $RepoRoot "harnesses") "opencode") "agents") "*.md"
Sync-HarnessAgents "codex" $codexRoot (Join-Path (Join-Path (Join-Path $RepoRoot "harnesses") "codex") "agents") "*.toml"

if ($Uninstall) {
    Remove-KarlSkills
}

if (-not $DryRun -and (Test-Path -LiteralPath $BackupRoot)) {
    Write-Host "Backups: $BackupRoot"
}
