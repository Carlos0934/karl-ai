[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($env:OPENCODE_API_KEY)) {
    Write-Host "SKIP: OPENCODE_API_KEY is not set."
    return
}

$RepoRoot = (Resolve-Path (Join-Path (Join-Path $PSScriptRoot "..") "..")).Path
$TestResults = Join-Path $RepoRoot "TestResults"
$TestRoot = Join-Path ([System.IO.Path]::GetTempPath()) "karl-opencode-e2e-$([guid]::NewGuid().ToString('N'))"
$TestHome = Join-Path $TestRoot "home"
$WorkTree = Join-Path $TestRoot "worktree"
$ArtifactName = "verified-artifact.txt"
$ArtifactContent = "karl-e2e-verified"
$StdoutPath = Join-Path $TestResults "opencode-stdout.jsonl"
$StderrPath = Join-Path $TestResults "opencode-stderr.log"
$environmentNames = @("HOME", "USERPROFILE", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "OPENCODE_DISABLE_EXTERNAL_SKILLS")
$originalEnvironment = @{}

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) { throw $Message }
}

try {
    New-Item -ItemType Directory -Path $TestResults, $TestHome, $WorkTree -Force | Out-Null
    Remove-Item -LiteralPath $StdoutPath, $StderrPath -Force -ErrorAction SilentlyContinue
    $openCodeConfig = Join-Path (Join-Path $TestHome ".config") "opencode"
    $agentsRoot = Join-Path $TestHome ".agents"
    New-Item -ItemType Directory -Path $openCodeConfig, (Join-Path $TestHome ".codex"), $agentsRoot -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $RepoRoot "skills") -Destination (Join-Path $agentsRoot "skills") -Recurse
    & (Join-Path $RepoRoot "install.ps1") -TargetHome $TestHome -SkipDeveloperModeCheck

    foreach ($name in $environmentNames) {
        $originalEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
    }
    $env:HOME = $TestHome
    $env:USERPROFILE = $TestHome
    $env:XDG_CONFIG_HOME = Join-Path $TestHome ".config"
    $env:XDG_DATA_HOME = Join-Path (Join-Path $TestHome ".local") "share"
    $env:XDG_CACHE_HOME = Join-Path $TestHome ".cache"
    $env:XDG_STATE_HOME = Join-Path (Join-Path $TestHome ".local") "state"
    Remove-Item Env:OPENCODE_DISABLE_EXTERNAL_SKILLS -ErrorAction SilentlyContinue

    & git init --quiet $WorkTree
    & git -C $WorkTree config user.name "Karl E2E"
    & git -C $WorkTree config user.email "karl-e2e@example.invalid"
    & git -C $WorkTree commit --quiet --allow-empty -m "test fixture"
    Assert-True ($LASTEXITCODE -eq 0) "Failed to initialize the temporary git repository."

    $prompt = @"
Use the controlled development workflow for this non-trivial behavioral task. Delegate implementation to karl-implementer and independent verification to karl-verifier. Create exactly one file named $ArtifactName in the repository root. Its content must be exactly '$ArtifactContent' with no newline and no BOM; use a byte-exact API. Do not create or modify any other repository file. Finish with the workflow state COMPLETED only after verifier PASS.
"@
    $openCode = (Get-Command opencode -ErrorAction Stop).Source
    & $openCode run $prompt --dir $WorkTree --agent karl-main --format json --auto > $StdoutPath 2> $StderrPath
    $exitCode = $LASTEXITCODE
    Assert-True ($exitCode -eq 0) "OpenCode exited with code $exitCode. See '$StderrPath'."

    $jsonLines = @(Get-Content -LiteralPath $StdoutPath | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    Assert-True ($jsonLines.Count -gt 0) "OpenCode produced no JSON events."
    $events = @($jsonLines | ForEach-Object { $_ | ConvertFrom-Json })
    $taskAgents = @($events | Where-Object {
        $_.type -eq "tool_use" -and $_.part.tool -eq "task" -and $_.part.state.status -eq "completed"
    } | ForEach-Object { $_.part.state.input.subagent_type })
    Assert-True ($taskAgents -ccontains "karl-implementer") "No completed task event shows that karl-implementer ran."
    Assert-True ($taskAgents -ccontains "karl-verifier") "No completed task event shows that karl-verifier ran."
    $assistantTextEvents = @($events | Where-Object { $_.type -eq "text" })
    Assert-True ($assistantTextEvents.Count -gt 0) "OpenCode produced no assistant text events."
    Assert-True ($assistantTextEvents[-1].part.text -cmatch "\bCOMPLETED\b") "The final assistant text event does not report workflow state COMPLETED."

    $artifact = Join-Path $WorkTree $ArtifactName
    Assert-True (Test-Path -LiteralPath $artifact -PathType Leaf) "The requested artifact was not created."
    $actualBytes = [System.IO.File]::ReadAllBytes($artifact)
    $expectedBytes = [System.Text.UTF8Encoding]::new($false).GetBytes($ArtifactContent)
    Assert-True ([Convert]::ToHexString($actualBytes) -ceq [Convert]::ToHexString($expectedBytes)) "The artifact content is not byte-exact."
    $status = @(& git -C $WorkTree status --porcelain=v1 --untracked-files=all)
    Assert-True (($status.Count -eq 1) -and ($status[0] -ceq "?? $ArtifactName")) "Unexpected persistent repository files: $($status -join ', ')"

    Write-Host "PASS: OpenCode ran implementer and verifier and produced the exact artifact."
} finally {
    foreach ($name in $environmentNames) {
        $value = $originalEnvironment[$name]
        if ($null -eq $value) {
            Remove-Item "Env:$name" -ErrorAction SilentlyContinue
        } else {
            [Environment]::SetEnvironmentVariable($name, $value, "Process")
        }
    }
    if (Test-Path -LiteralPath $TestRoot) {
        Remove-Item -LiteralPath $TestRoot -Recurse -Force
    }
}
