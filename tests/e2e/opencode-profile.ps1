[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path (Join-Path $PSScriptRoot "..") "..")).Path
$Applier = Join-Path $RepoRoot "scripts/apply-opencode-profile.ps1"
$WorkRoot = Join-Path ([System.IO.Path]::GetTempPath()) "karl-profile-e2e-$([guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $WorkRoot -Force | Out-Null

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) { throw $Message }
}

function Invoke-Applier {
    # Runs the applier as a child pwsh process so its exit code is reliable
    # ($LASTEXITCODE goes stale for in-process .ps1 calls that end without
    # an explicit exit statement).
    param([string[]]$ArgumentList)

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        & pwsh -NoProfile -File $Applier @ArgumentList 2>&1 | Out-Null
        return $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previous
    }
}

function Get-JsonText {
    param([string]$Path)
    return [System.IO.File]::ReadAllText($Path)
}

try {
    # --- 1. Profile listing ---------------------------------------------------
    $listed = & $Applier -List
    Assert-True ($listed -contains "karl-default") "Profile list is missing 'karl-default'."
    Assert-True ($listed -contains "openai") "Profile list is missing 'openai'."
    Assert-True ($listed -contains "contributor") "Profile list is missing 'contributor'."

    # --- 2. Real profiles parse ------------------------------------------------
    foreach ($name in @("karl-default", "openai", "contributor")) {
        $code = Invoke-Applier @("-Profile", $name, "-Scope", "project", "-TargetDir", (Join-Path $WorkRoot "dry-$name"), "-DryRun")
        Assert-True ($code -eq 0) "Dry run of profile '$name' failed with exit $code."
        Assert-True (-not (Test-Path -LiteralPath (Join-Path $WorkRoot "dry-$name/opencode.json"))) "Dry run of '$name' wrote a file."
    }

    # --- 3. Unknown profile fails without writing ------------------------------
    $unknownDir = Join-Path $WorkRoot "unknown"
    $code = Invoke-Applier @("-Profile", "no-such-profile", "-Scope", "project", "-TargetDir", $unknownDir)
    Assert-True ($code -eq 2) "Unknown profile should exit 2 (got $code)."
    Assert-True (-not (Test-Path -LiteralPath (Join-Path $unknownDir "opencode.json"))) "Unknown profile wrote a file."

    # --- 4. Merge preserves unrelated keys, overwrites stale karl models -------
    $synthetic = [ordered]@{
        "karl-orchestrator" = "test-provider/test-orch#medium"
        "karl-worker"       = "test-provider/test-work#xhigh"
        "karl-scout"        = "test-provider/test-scout#high"
        "karl-verify"       = "test-provider/test-verify#high"
        "karl-reviewer"     = "test-provider/test-review#high"
    }
    $syntheticPath = Join-Path $WorkRoot "synthetic.json"
    [System.IO.File]::WriteAllText($syntheticPath, ($synthetic | ConvertTo-Json -Depth 4))

    $projectDir = Join-Path $WorkRoot "project"
    New-Item -ItemType Directory -Path $projectDir -Force | Out-Null
    $existing = [ordered]@{
        '$schema' = "https://opencode.ai/config.json"
        model     = "some-provider/some-model"
        mcp       = [ordered]@{ jira = [ordered]@{ type = "remote"; enabled = $false } }
        agent     = [ordered]@{
            build         = [ordered]@{ mode = "primary"; model = "some-provider/other" }
            "karl-worker" = [ordered]@{ mode = "subagent"; model = "stale-provider/stale" }
        }
    }
    $dest = Join-Path $projectDir "opencode.json"
    [System.IO.File]::WriteAllText($dest, ($existing | ConvertTo-Json -Depth 8))

    $code = Invoke-Applier @("-Profile", $syntheticPath, "-Scope", "project", "-TargetDir", $projectDir)
    Assert-True ($code -eq 0) "Profile apply failed with exit $code."
    $merged = Get-JsonText $dest | ConvertFrom-Json
    Assert-True ($merged.model -eq "some-provider/some-model") "Global model was not preserved."
    Assert-True ($merged.mcp.jira.enabled -eq $false) "Unrelated mcp config was not preserved."
    Assert-True ($merged.agent.build.model -eq "some-provider/other") "Non-karl agent was not preserved."
    Assert-True ($merged.agent."karl-worker".model -eq "test-provider/test-work#xhigh") "Stale karl-worker model was not overwritten."
    Assert-True ($merged.agent."karl-worker".mode -eq "subagent") "Existing karl-worker fields were dropped."
    Assert-True ($merged.agent."karl-orchestrator".model -eq "test-provider/test-orch#medium") "karl-orchestrator was not added."
    Assert-True ($merged.agent."karl-reviewer".model -eq "test-provider/test-review#high") "karl-reviewer was not added."
    $backups = @(Get-ChildItem -LiteralPath $projectDir -Filter "opencode.json.bak-*")
    Assert-True ($backups.Count -eq 1) "Expected exactly one backup, found $($backups.Count)."

    # --- 5. Idempotence: second run changes nothing, backs up nothing -----------
    $before = Get-JsonText $dest
    $code = Invoke-Applier @("-Profile", $syntheticPath, "-Scope", "project", "-TargetDir", $projectDir)
    Assert-True ($code -eq 0) "Idempotent re-apply failed with exit $code."
    Assert-True ((Get-JsonText $dest) -eq $before) "Idempotent re-apply rewrote the file."
    Assert-True (@(Get-ChildItem -LiteralPath $projectDir -Filter "opencode.json.bak-*").Count -eq 1) "Idempotent re-apply created a backup."

    # --- 6. --force skips the backup -------------------------------------------
    $synthetic["karl-scout"] = "test-provider/test-scout-v2#high"
    [System.IO.File]::WriteAllText($syntheticPath, ($synthetic | ConvertTo-Json -Depth 4))
    $code = Invoke-Applier @("-Profile", $syntheticPath, "-Scope", "project", "-TargetDir", $projectDir, "-Force")
    Assert-True ($code -eq 0) "Forced apply failed with exit $code."
    Assert-True (@(Get-ChildItem -LiteralPath $projectDir -Filter "opencode.json.bak-*").Count -eq 1) "--force still created a backup."
    $merged = Get-JsonText $dest | ConvertFrom-Json
    Assert-True ($merged.agent."karl-scout".model -eq "test-provider/test-scout-v2#high") "--force did not apply the new model."

    # --- 7. Global scope writes under the home ----------------------------------
    $fakeHome = Join-Path $WorkRoot "home"
    $code = Invoke-Applier @("-Profile", "openai", "-Scope", "global", "-HomeDir", $fakeHome)
    Assert-True ($code -eq 0) "Global apply failed with exit $code."
    $globalDest = Join-Path $fakeHome ".config/opencode/opencode.json"
    Assert-True (Test-Path -LiteralPath $globalDest) "Global apply did not create '$globalDest'."
    $merged = Get-JsonText $globalDest | ConvertFrom-Json
    Assert-True ($merged.agent."karl-orchestrator".model -eq "openai/gpt-6-sol#medium") "Global karl-orchestrator mismatch."
    Assert-True ($merged.agent."karl-worker".model -eq "openai/gpt-6-luna#xhigh") "Global karl-worker mismatch."
    Assert-True ($merged.agent."karl-scout".model -eq "openai/gpt-6-luna#high") "Global karl-scout mismatch."
    Assert-True ($merged.agent."karl-reviewer".model -eq "openai/gpt-6-sol#high") "Global karl-reviewer mismatch."

    # --- 8. Invalid inputs fail closed -------------------------------------------
    $badProfile = Join-Path $WorkRoot "bad.json"
    [System.IO.File]::WriteAllText($badProfile, "{not json")
    $badDir = Join-Path $WorkRoot "bad"
    $code = Invoke-Applier @("-Profile", $badProfile, "-Scope", "project", "-TargetDir", $badDir)
    Assert-True ($code -ne 0) "Invalid profile JSON should fail."

    $evilProfile = Join-Path $WorkRoot "evil.json"
    [System.IO.File]::WriteAllText($evilProfile, '{"build": {"model": "x/y"}}')
    $evilDir = Join-Path $WorkRoot "evil"
    $code = Invoke-Applier @("-Profile", $evilProfile, "-Scope", "project", "-TargetDir", $evilDir)
    Assert-True ($code -ne 0) "Non-karl profile entry should fail."

    $badDestDir = Join-Path $WorkRoot "baddest"
    New-Item -ItemType Directory -Path $badDestDir -Force | Out-Null
    [System.IO.File]::WriteAllText((Join-Path $badDestDir "opencode.json"), "[1,2]")
    $code = Invoke-Applier @("-Profile", $syntheticPath, "-Scope", "project", "-TargetDir", $badDestDir)
    Assert-True ($code -ne 0) "Non-object destination should fail."
    Assert-True ((Get-JsonText (Join-Path $badDestDir "opencode.json")) -eq "[1,2]") "Failed apply mutated the destination."

    # --- 9. Remote profile via -ProfileUrl (file:// keeps it offline) ------------
    $urlDir = Join-Path $WorkRoot "url-project"
    $fileUrl = "file:///" + ($syntheticPath -replace '\\', '/')
    $code = Invoke-Applier @("-ProfileUrl", $fileUrl, "-Scope", "project", "-TargetDir", $urlDir)
    Assert-True ($code -eq 0) "Profile URL apply failed with exit $code."
    $urlDest = Join-Path $urlDir "opencode.json"
    $merged = Get-JsonText $urlDest | ConvertFrom-Json
    Assert-True ($merged.agent."karl-worker".model -eq "test-provider/test-work#xhigh") "Profile URL karl-worker mismatch."
    Assert-True ($merged.agent."karl-orchestrator".model -eq "test-provider/test-orch#medium") "Profile URL karl-orchestrator mismatch."

    $missingUrlDir = Join-Path $WorkRoot "url-missing"
    $code = Invoke-Applier @("-ProfileUrl", "file:///no-such-profile-here.json", "-Scope", "project", "-TargetDir", $missingUrlDir)
    Assert-True ($code -ne 0) "Unreachable profile URL should fail."
    Assert-True (-not (Test-Path -LiteralPath (Join-Path $missingUrlDir "opencode.json"))) "Failed URL apply wrote a file."

    Write-Host "PASS: opencode profile apply is scoped, preserving, idempotent, and fails closed."
} finally {
    Remove-Item -LiteralPath $WorkRoot -Recurse -Force -ErrorAction SilentlyContinue
}
