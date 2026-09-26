[CmdletBinding()]
param(
    [string]$Profile = "karl-default",
    [string]$ProfileUrl = "",
    [ValidateSet("project", "global")][string]$Scope = "project",
    [string]$TargetDir = "",
    [string]$HomeDir = "",
    [switch]$DryRun,
    [switch]$Force,
    [switch]$List
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-ProfilesDir {
    if ([string]::IsNullOrEmpty($PSScriptRoot)) {
        throw "No checkout context (script streamed from remote). Use -Profile with a file path or -ProfileUrl <url>."
    }
    $dir = Join-Path (Split-Path -Parent $PSScriptRoot) "profiles"
    if (-not (Test-Path -LiteralPath $dir -PathType Container)) {
        throw "Profiles directory not found at '$dir'."
    }
    return $dir
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

function Get-ProfileMap {
    param([string]$Name, [string]$Url)

    # The human-readable source label used in log messages.
    $script:profileLabel = if ($Url) { $Url } else { $Name }
    $downloaded = $null
    try {
        if ($Url) {
            if ($Url -notmatch '^(https?|file)://') {
                throw "Profile URL must start with https:// or file:// (found '$Url')."
            }
            $downloaded = Join-Path ([System.IO.Path]::GetTempPath()) ("karl-profile-" + [guid]::NewGuid().ToString('N') + ".json")
            if ($Url.StartsWith("file://")) {
                Copy-Item -LiteralPath (([System.Uri]$Url).LocalPath) -Destination $downloaded -Force
            } else {
                Invoke-WebRequest -Uri $Url -OutFile $downloaded
            }
            $profilePath = $downloaded
        } elseif ($Name -match '[/\\]|\.json$') {
            $profilePath = $Name
        } else {
            $profilesDir = Get-ProfilesDir
            $profilePath = Join-Path $profilesDir "$Name.json"
            if (-not (Test-Path -LiteralPath $profilePath -PathType Leaf)) {
                Write-Host "apply-opencode-profile: unknown profile '$Name'." -ForegroundColor Red
                Write-Host "Available profiles in '$profilesDir':"
                foreach ($file in (Get-ChildItem -LiteralPath $profilesDir -Filter "*.json" -File -ErrorAction SilentlyContinue)) {
                    Write-Host ("  " + [System.IO.Path]::GetFileNameWithoutExtension($file.Name))
                }
                exit 2
            }
        }
        try {
            $parsed = [System.IO.File]::ReadAllText($profilePath) | ConvertFrom-Json
        } catch {
            throw "Profile '$profilePath' is not valid JSON: $($_.Exception.Message)"
        }
        $map = @{}
        foreach ($property in $parsed.PSObject.Properties) {
            if ($property.Name -notmatch '^karl-') {
                throw "Profile '$profilePath' must only define 'karl-*' agents (found '$($property.Name)')."
            }
            if ($property.Value -isnot [string] -or $property.Value -notmatch '^[^/\s]+/\S+$') {
                throw "Profile '$profilePath' entry '$($property.Name)' must be '<provider>/<model>' (found '$($property.Value)')."
            }
            $map[$property.Name] = $property.Value
        }
        if ($map.Count -eq 0) {
            throw "Profile '$profilePath' defines no agent models."
        }
        return $map
    } finally {
        if ($downloaded -and (Test-Path -LiteralPath $downloaded)) {
            Remove-Item -LiteralPath $downloaded -Force -ErrorAction SilentlyContinue
        }
    }
}

if ($List) {
    $profilesDir = Get-ProfilesDir
    foreach ($file in (Get-ChildItem -LiteralPath $profilesDir -Filter "*.json" -File -ErrorAction SilentlyContinue)) {
        Write-Output ([System.IO.Path]::GetFileNameWithoutExtension($file.Name))
    }
    exit 0
}

$profileMap = Get-ProfileMap $Profile $ProfileUrl

if ($Scope -eq "global") {
    $homeBase = if ($HomeDir) { $HomeDir } else { $HOME }
    $destination = Join-Path $homeBase ".config/opencode/opencode.json"
} else {
    $base = if ($TargetDir) { $TargetDir } else { (Get-Location).Path }
    $destination = Join-Path ([System.IO.Path]::GetFullPath($base)) "opencode.json"
}

$config = $null
$destinationExisted = Test-Path -LiteralPath $destination -PathType Leaf
if ($destinationExisted) {
    try {
        $config = [System.IO.File]::ReadAllText($destination) | ConvertFrom-Json
    } catch {
        throw "Destination '$destination' is not valid JSON: $($_.Exception.Message)"
    }
    if ($config -isnot [System.Management.Automation.PSObject] -or $config -is [System.Array]) {
        throw "Destination '$destination' must hold a JSON object."
    }
} else {
    $config = [pscustomobject]@{ '$schema' = "https://opencode.ai/config.json" }
}

$agentProp = $config.PSObject.Properties['agent']
if ($null -eq $agentProp) {
    $config | Add-Member -NotePropertyName "agent" -NotePropertyValue ([pscustomobject]@{})
    $agent = $config.PSObject.Properties['agent'].Value
} else {
    $agent = $agentProp.Value
    if ($agent -isnot [System.Management.Automation.PSObject] -or $agent -is [System.Array]) {
        throw "Destination '$destination' has a non-object 'agent' key; refusing to merge."
    }
}

$changes = @()
foreach ($name in ($profileMap.Keys | Sort-Object)) {
    $wanted = $profileMap[$name]
    $entry = $agent.PSObject.Properties[$name]
    if ($null -eq $entry) {
        $agent | Add-Member -NotePropertyName $name -NotePropertyValue ([pscustomobject]@{ model = $wanted })
        $changes += [pscustomobject]@{ Name = $name; Before = "(absent)"; After = $wanted }
    } else {
        if ($entry.Value -isnot [System.Management.Automation.PSObject] -or $entry.Value -is [System.Array]) {
            throw "Destination '$destination' has a non-object 'agent.$name' entry; refusing to merge."
        }
        $before = $entry.Value.model
        if ($before -ne $wanted) {
            $entry.Value | Add-Member -NotePropertyName "model" -NotePropertyValue $wanted -Force
            if ($null -eq $before) { $before = "(absent)" }
            $changes += [pscustomobject]@{ Name = $name; Before = $before; After = $wanted }
        }
    }
}

if ($changes.Count -eq 0) {
    Write-Action "Profile '$profileLabel' already applied to '$destination' ($($profileMap.Count) models current)."
    exit 0
}

foreach ($change in $changes) {
    Write-Host "$($change.Name): $($change.Before) -> $($change.After)"
}

if (-not $DryRun) {
    $parent = Split-Path -Parent $destination
    if ($parent -and -not (Test-Path -LiteralPath $parent)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    if ($destinationExisted -and -not $Force) {
        $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
        $backupPath = "$destination.bak-$stamp"
        $suffix = 2
        while (Test-Path -LiteralPath $backupPath) {
            $backupPath = "$destination.bak-$stamp-$suffix"
            $suffix++
        }
        Copy-Item -LiteralPath $destination -Destination $backupPath -Force
        Write-Action "Back up $destination -> $backupPath"
    }
    Invoke-Mutation "Apply profile '$profileLabel' ($($changes.Count) models) to '$destination'" {
        [System.IO.File]::WriteAllText($destination, ($config | ConvertTo-Json -Depth 16))
    }
} else {
    Write-Action "Profile '$profileLabel' would apply $($changes.Count) models to '$destination'."
}
exit 0
