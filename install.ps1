[CmdletBinding()]
param(
    [switch]$DryRun,
    [switch]$Force,
    [switch]$Uninstall,
    [string]$TargetHome = $HOME,
    [string]$RepoUrl = "",
    [switch]$SkipDeveloperModeCheck
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Single repository-root variable for the whole flow. In bootstrap mode
# (-RepoUrl) Update-RepoClone rebinds it to the clone directory before any
# later use; every consumer below must read this variable only.
$RepoRoot = $PSScriptRoot
$TargetHome = [System.IO.Path]::GetFullPath($TargetHome)
$MarkerStart = "<!-- karl-ai: controlled-development -->"
$MarkerEnd = "<!-- /karl-ai: controlled-development -->"
$ManagedPattern = "(?s)$([regex]::Escape($MarkerStart)).*?$([regex]::Escape($MarkerEnd))"
$BackupRoot = Join-Path (Join-Path $TargetHome ".agents-backup") (Get-Date -Format 'yyyyMMdd-HHmmss')
$PathComparison = if ($IsWindows) { [System.StringComparison]::OrdinalIgnoreCase } else { [System.StringComparison]::Ordinal }
$DirectorySeparators = [char[]]@([System.IO.Path]::DirectorySeparatorChar, [System.IO.Path]::AltDirectorySeparatorChar)

# Bootstrap detection. The installer only runs in place when the script
# directory looks like the real repository layout: AGENTS.md plus the skills
# and harnesses directories. When it has no script file context (irm/iex,
# scriptblock, stdin: $PSScriptRoot is empty) or sits outside such a layout
# (for example a copied installer beside a stray AGENTS.md), the repository
# source must come from -RepoUrl.
$bootstrapMode = [string]::IsNullOrEmpty($PSScriptRoot) -or
    -not (Test-Path -LiteralPath (Join-Path $PSScriptRoot "AGENTS.md") -PathType Leaf) -or
    -not (Test-Path -LiteralPath (Join-Path $PSScriptRoot "skills") -PathType Container) -or
    -not (Test-Path -LiteralPath (Join-Path $PSScriptRoot "harnesses") -PathType Container)
if ($bootstrapMode -and [string]::IsNullOrWhiteSpace($RepoUrl)) {
    throw "-RepoUrl <url-or-path> is required when the installer does not run from a checkout (for example dot-downloaded)."
}

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

function Get-NormalizedGitUrl {
    # Strips trailing separators and a trailing ".git"; for local paths also
    # normalizes separators, because git may store a local clone origin with
    # Windows- or POSIX-style separators depending on how it was given.
    param([string]$Url)

    $normalized = $Url.TrimEnd('/', '\')
    if ($normalized.EndsWith(".git", [System.StringComparison]::OrdinalIgnoreCase)) {
        $normalized = $normalized.Substring(0, $normalized.Length - 4)
    }
    $normalized = $normalized.TrimEnd('/', '\')
    if ($normalized -notmatch "://" -and -not $normalized.StartsWith("git@")) {
        $normalized = $normalized.Replace('\', '/')
    }
    return $normalized.ToLowerInvariant()
}

function Invoke-GitCapture {
    # Runs git and captures stdout lines, swallowing stderr; returns the exit
    # code and the captured stdout lines. ErrorActionPreference is relaxed
    # around the native call so redirected stderr cannot trip EAP=Stop.
    param([string[]]$GitArguments)

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $output = & git @GitArguments 2>&1
        $lines = @($output | ForEach-Object { if ($_ -is [string]) { $_ } })
        return [pscustomobject]@{ ExitCode = $LASTEXITCODE; Lines = $lines }
    } finally {
        $ErrorActionPreference = $previous
    }
}

function Invoke-GitVisible {
    # Runs git showing its output; returns only the exit code. Git's combined
    # output is re-printed via Write-Host so it cannot leak into the return
    # value.
    param([string[]]$GitArguments)

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $output = & git @GitArguments 2>&1
        $exitCode = $LASTEXITCODE
        foreach ($line in @($output)) {
            Write-Host "$line"
        }
        return $exitCode
    } finally {
        $ErrorActionPreference = $previous
    }
}

function Invoke-RepoClone {
    # Clones $Url into $Dir. Shallow by default; a local path that rejects
    # --depth 1 is retried as a normal clone.
    param(
        [string]$Url,
        [string]$Dir,
        [bool]$IsRemote
    )

    $exitCode = Invoke-GitVisible @("clone", "--depth", "1", $Url, $Dir)
    if ($exitCode -eq 0) { return }
    if ($IsRemote) { throw "git clone failed for '$Url'." }
    Write-Warning "Retrying clone of '$Url' without --depth 1."
    $exitCode = Invoke-GitVisible @("clone", $Url, $Dir)
    if ($exitCode -ne 0) { throw "git clone failed for '$Url'." }
}

function Invoke-CloneIntoPlace {
    # Clones $Url into a temporary directory next to $Dir and moves it into
    # place, so a concurrent run can never observe a partial clone at $Dir.
    # The lock (see Acquire-InstallLock) must be held by the caller.
    param(
        [string]$Url,
        [string]$Dir,
        [bool]$IsRemote
    )

    $parent = Split-Path -Parent $Dir
    $tmpDir = Join-Path $parent ("$([System.IO.Path]::GetFileName($Dir)).tmp-$PID-$(Get-Date -Format 'yyyyMMdd-HHmmss')")
    if (Test-Path -LiteralPath $tmpDir) {
        Remove-Item -LiteralPath $tmpDir -Recurse -Force
    }
    try {
        Invoke-RepoClone -Url $Url -Dir $tmpDir -IsRemote $IsRemote
        if (Test-Path -LiteralPath $Dir) {
            throw "Destination '$Dir' appeared while cloning; refusing to overwrite it."
        }
        Move-Item -LiteralPath $tmpDir -Destination $Dir
    } catch {
        if (Test-Path -LiteralPath $tmpDir) {
            Remove-Item -LiteralPath $tmpDir -Recurse -Force
        }
        throw
    }
}

function Acquire-InstallLock {
    # Creates the advisory install lock file with create-new (O_EXCL)
    # semantics and keeps an exclusive handle on it for the whole
    # clone-or-pull-then-install sequence. If the lock exists, retries
    # briefly, then fails with a clear message. Returns the FileStream; the
    # caller must delete the file and dispose the handle to release the lock.
    param([string]$Path)

    $null = New-Item -ItemType Directory -Path (Split-Path -Parent $Path) -Force
    $deadline = [DateTime]::UtcNow.AddSeconds(15)
    while ($true) {
        try {
            return [System.IO.File]::Open($Path, [System.IO.FileMode]::CreateNew, [System.IO.FileAccess]::ReadWrite, [System.IO.FileShare]::Delete)
        } catch [System.IO.IOException] {
            if ([DateTime]::UtcNow -ge $deadline) {
                throw "Another install is running or left a stale lock at '$Path'. Remove it if no install is active, then retry."
            }
            Start-Sleep -Milliseconds 500
        }
    }
}

function Update-RepoClone {
    # Clones or updates the repository at $Dir from $Url and returns the
    # repository root directory to use for the rest of the install: the clone
    # itself. The repo location follows -TargetHome (the config home) so tests
    # can use an isolated home; in real installs -TargetHome defaults to $HOME,
    # making $HOME/.agents the canonical repo location. Re-running updates the
    # clone with git pull --ff-only.
    param(
        [string]$Url,
        [string]$Dir
    )

    $wanted = Get-NormalizedGitUrl $Url
    $isRemote = ($Url -match "://") -or $Url.StartsWith("git@")

    if (-not (Test-Path -LiteralPath $Dir)) {
        if ($DryRun) {
            # Nothing else can be previewed: the canonical files live in the
            # clone that only a real run creates.
            Write-Action "Clone $Url into $Dir"
            exit 0
        }
        Write-Action "Clone $Url into $Dir"
        Invoke-CloneIntoPlace -Url $Url -Dir $Dir -IsRemote $isRemote
        return $Dir
    }

    $isWorkTree = $false
    $origin = $null
    if (Test-Path -LiteralPath (Join-Path $Dir ".git")) {
        $inside = Invoke-GitCapture @("-C", $Dir, "rev-parse", "--is-inside-work-tree")
        $isWorkTree = ($inside.ExitCode -eq 0) -and ($inside.Lines.Count -gt 0) -and ($inside.Lines[0].Trim() -eq "true")
        if ($isWorkTree) {
            $originResult = Invoke-GitCapture @("-C", $Dir, "remote", "get-url", "origin")
            if ($originResult.ExitCode -eq 0 -and $originResult.Lines.Count -gt 0) {
                $origin = $originResult.Lines[0].Trim()
            }
        }
    }

    $matchesOrigin = $false
    if ($origin) {
        $current = Get-NormalizedGitUrl $origin
        if ($current -ieq $wanted) {
            $matchesOrigin = $true
        } elseif (-not $isRemote) {
            $wantedAbsolute = [System.IO.Path]::GetFullPath($Url)
            if ((Get-NormalizedGitUrl $wantedAbsolute) -ieq $current) {
                $matchesOrigin = $true
            }
        }
    }

    if ($matchesOrigin) {
        Write-Action "Update repo at $Dir (git pull --ff-only)"
        if (-not $DryRun) {
            $exitCode = Invoke-GitVisible @("-C", $Dir, "pull", "--ff-only")
            if ($exitCode -ne 0) { throw "git pull --ff-only failed in '$Dir'." }
        }
        return $Dir
    }

    if (-not $Force) {
        throw "'$Dir' exists but is not a git clone of '$Url'. Re-run with -Force to back it up and re-clone."
    }

    if ($DryRun) {
        Write-Action "Back up '$Dir' to $(Join-Path $BackupRoot 'agents') and clone '$Url' into '$Dir'"
        # Nothing else can be previewed: the canonical files live in the clone
        # that only a real run creates.
        exit 0
    }

    Write-Warning "Backing up '$Dir' to $(Join-Path $BackupRoot 'agents') and re-cloning '$Url'."
    New-Item -ItemType Directory -Path $BackupRoot -Force | Out-Null
    Move-Item -LiteralPath $Dir -Destination (Join-Path $BackupRoot "agents")
    Invoke-CloneIntoPlace -Url $Url -Dir $Dir -IsRemote $isRemote
    return $Dir
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

# The advisory install lock serializes the whole clone-or-pull-then-install
# sequence against concurrent runs on the same target home. It is only taken
# on the -RepoUrl path (the only path that touches <target>/.agents), and
# never on a dry run. It is released in the finally block on success and
# failure alike.
$lockPath = Join-Path $TargetHome ".agents.install.lock"
$lockHandle = $null
try {
    if (-not [string]::IsNullOrWhiteSpace($RepoUrl) -and -not $DryRun) {
        $lockHandle = Acquire-InstallLock -Path $lockPath
    }

    if (-not [string]::IsNullOrWhiteSpace($RepoUrl)) {
        # Bind $RepoRoot to the clone for the ENTIRE remaining flow (canonical
        # AGENTS.md, skills validation, harness link sources), whether or not
        # the script has a usable $PSScriptRoot.
        $RepoRoot = Update-RepoClone -Url $RepoUrl -Dir (Join-Path $TargetHome ".agents")
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
} finally {
    if ($lockHandle) {
        # Delete before dispose: the handle is opened with FileShare.Delete so
        # the file is removed atomically with the handle release and cannot
        # outlive it (no stale-lock race with a waiting second run).
        Remove-Item -LiteralPath $lockPath -Force -ErrorAction SilentlyContinue
        $lockHandle.Dispose()
    }
}
