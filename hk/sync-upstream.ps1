#Requires -Version 5.1
<#
.SYNOPSIS
  Preview or start a merge of Wei-Shaw/sub2api into this fork.

.DESCRIPTION
  Fetches upstream through the local proxy, then either previews conflicts with
  git merge-tree (-DryRun) or starts an uncommitted merge. Paths listed in
  hk/upstream-removed.json are deleted when they conflict. The script does not
  commit or push. Rename detection is disabled so a moved fork file is not
  treated as the place to apply upstream edits.

.PARAMETER DryRun
  Fetch and write a report only. Does not create a branch or change the index.
#>
param([switch]$DryRun)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

function Invoke-Git {
  param([Parameter(ValueFromRemainingArguments = $true)][string[]]$GitArgs)
  & git @GitArgs
  if ($LASTEXITCODE -ne 0) {
    throw "git $($GitArgs -join ' ') failed with exit $LASTEXITCODE"
  }
}

function Get-ConflictPath([string]$Line) {
  if ($Line -match 'Merge conflict in (.+)$') { return $Matches[1].Trim() }
  if ($Line -match 'CONFLICT \([^)]+\): (.+) deleted in .+$') { return $Matches[1].Trim() }
  if ($Line -match 'CONFLICT \([^)]+\): ([^\s]+) ') { return $Matches[1].Trim() }
  return $null
}

function Test-RemovedPath([string]$Path, $Entries) {
  foreach ($entry in $Entries) {
    $pattern = [string]$entry.path
    if ($pattern.EndsWith('/**')) {
      $prefix = $pattern.Substring(0, $pattern.Length - 3)
      if ($Path -eq $prefix.TrimEnd('/') -or $Path.StartsWith($prefix)) { return $true }
    } elseif ($Path -eq $pattern) {
      return $true
    }
  }
  return $false
}

$dirty = @(git status --porcelain)
if (-not $DryRun -and $dirty.Count -gt 0) {
  throw 'Working tree is not clean. Commit or stash before merging. -DryRun can still preview.'
}

$env:HTTP_PROXY = 'http://127.0.0.1:7890'
$env:HTTPS_PROXY = 'http://127.0.0.1:7890'
$env:GIT_TERMINAL_PROMPT = '0'
Write-Host 'Fetching upstream through the local proxy...'
Invoke-Git fetch upstream

$manifestPath = Join-Path $repoRoot 'hk/upstream-removed.json'
$manifest = Get-Content -Raw -Path $manifestPath | ConvertFrom-Json
$removed = @($manifest.removed)
$version = (git show 'upstream/main:backend/cmd/server/VERSION' | Out-String).Trim()
if (-not $version) { $version = 'unknown' }
$upstreamSha = (git rev-parse --short upstream/main).Trim()
$headSha = (git rev-parse --short HEAD).Trim()
$base = (git merge-base HEAD upstream/main).Trim()
$date = Get-Date -Format 'yyyy-MM-dd'
$reportDir = Join-Path $repoRoot 'hk/reports'
New-Item -ItemType Directory -Force -Path $reportDir | Out-Null
$reportPath = Join-Path $reportDir "$date-$upstreamSha.md"

$mergeOutput = & git -c merge.renames=false merge-tree --write-tree --name-only --messages HEAD upstream/main 2>&1 | Out-String
$conflictPaths = New-Object System.Collections.Generic.List[string]
foreach ($line in ($mergeOutput -split "`r?`n")) {
  $path = Get-ConflictPath $line
  if ($path -and -not $conflictPaths.Contains($path)) { $conflictPaths.Add($path) }
}

$autoRemoved = @($conflictPaths | Where-Object { Test-RemovedPath $_ $removed })
$entConflicts = @($conflictPaths | Where-Object { $_ -like 'backend/ent/*' -and $_ -notlike 'backend/ent/schema/*' })
$wireConflicts = @($conflictPaths | Where-Object { $_ -like '*wire_gen.go' })
$manual = @($conflictPaths | Where-Object {
  -not (Test-RemovedPath $_ $removed) -and $_ -notlike 'backend/ent/*' -and $_ -notlike '*wire_gen.go'
})

if (-not $DryRun) {
  $backup = "codex/backup-pre-upstream-merge-$date-$headSha"
  $branch = "merge/upstream-$version-hk"
  if (git show-ref --verify --quiet "refs/heads/$branch") {
    throw "Branch $branch already exists."
  }
  Invoke-Git branch $backup
  Invoke-Git checkout -b $branch
  Invoke-Git config rerere.enabled true
  Invoke-Git config rerere.autoupdate true
  Write-Host "Merging upstream/main onto $branch without committing..."
  & git -c merge.renames=false merge --no-ff --no-commit upstream/main
  $mergeExit = $LASTEXITCODE
  if ($mergeExit -ne 0 -and $mergeExit -ne 1) {
    throw "git merge failed with exit $mergeExit"
  }
  foreach ($path in $autoRemoved) {
    if (Test-Path -LiteralPath $path) {
      & git rm -f -- $path
    } else {
      & git rm -f -- $path 2>$null
      if ($LASTEXITCODE -ne 0) { git add -u -- $path }
    }
  }
}

$addedUser = @(git diff --name-status --diff-filter=A $base upstream/main -- frontend/src/views/user frontend/src/api frontend/src/i18n)
$migrationLines = New-Object System.Collections.Generic.List[string]
$ourMigrations = @(git diff --name-only $base HEAD -- backend/migrations)
$theirMigrations = @(git diff --name-only $base upstream/main -- backend/migrations)
$ourPrefixes = @{}
foreach ($file in $ourMigrations) {
  $name = Split-Path -Leaf $file
  if ($name -match '^(\d+)_') { $ourPrefixes[$Matches[1]] = $true }
}
foreach ($file in $theirMigrations) {
  $name = Split-Path -Leaf $file
  if ($name -match '^(\d+)_' -and $ourPrefixes.ContainsKey($Matches[1])) {
    $migrationLines.Add("$($Matches[1]) overlaps: $name")
  }
}

$hotspots = @(git diff --numstat upstream/main -- frontend/src backend |
  Where-Object { $_ -and $_ -notmatch 'backend/ent/' } |
  ForEach-Object {
    $parts = $_ -split "`t"
    if ($parts.Count -ge 3 -and $parts[0] -match '^\d+$' -and $parts[1] -match '^\d+$') {
      [pscustomobject]@{ Added = [int]$parts[0]; Deleted = [int]$parts[1]; Path = $parts[2]; Churn = [int]$parts[0] + [int]$parts[1] }
    }
  } | Sort-Object Churn -Descending | Select-Object -First 15)

$sb = New-Object System.Text.StringBuilder
[void]$sb.AppendLine("# Upstream sync preview $date")
[void]$sb.AppendLine()
[void]$sb.AppendLine("- Mode: $(if ($DryRun) { 'dry run' } else { 'merge started, not committed' })")
[void]$sb.AppendLine("- HEAD: $headSha")
[void]$sb.AppendLine("- upstream/main: $upstreamSha ($version)")
[void]$sb.AppendLine("- merge-base: $base")
[void]$sb.AppendLine()
[void]$sb.AppendLine('## Conflicts')
[void]$sb.AppendLine()
if ($conflictPaths.Count -eq 0) {
  [void]$sb.AppendLine('No conflicts predicted.')
} else {
  foreach ($path in $conflictPaths) {
    $action = 'resolve by hand'
    if (Test-RemovedPath $path $removed) { $action = 'delete; port behavior from the section below' }
    elseif ($path -like 'backend/ent/*') { $action = 'keep schema edits, then run go generate ./ent' }
    elseif ($path -like '*wire_gen.go') { $action = 'run go generate ./cmd/server' }
    [void]$sb.AppendLine("- ``$path``: $action")
  }
}
[void]$sb.AppendLine()
[void]$sb.AppendLine('## Removed paths upstream changed')
[void]$sb.AppendLine()
$changedRemoved = @()
foreach ($entry in $removed) {
  $pattern = [string]$entry.path
  if ($pattern.EndsWith('/**')) { continue }
  $stat = git diff --shortstat $base upstream/main -- $pattern
  if ($stat) { $changedRemoved += [pscustomobject]@{ Path = $pattern; Stat = $stat.Trim(); Replacement = [string]$entry.replacement } }
}
if ($changedRemoved.Count -eq 0) {
  [void]$sb.AppendLine('Upstream did not change any listed file since the merge base.')
} else {
  foreach ($item in $changedRemoved) {
    $target = if ($item.Replacement) { $item.Replacement } else { 'no replacement' }
    [void]$sb.AppendLine("- ``$($item.Path)`` ($($item.Stat)) -> $target")
    $commits = git log --oneline "$base..upstream/main" -- $item.Path
    if ($commits) { [void]$sb.AppendLine('  commits: ' + (($commits | Select-Object -First 8) -join '; ')) }
  }
}
[void]$sb.AppendLine()
[void]$sb.AppendLine('## Upstream additions to review')
[void]$sb.AppendLine()
if ($addedUser.Count -eq 0) { [void]$sb.AppendLine('No added user views, API modules, or locale files.') }
else { foreach ($line in $addedUser) { [void]$sb.AppendLine("- $line") } }
[void]$sb.AppendLine()
[void]$sb.AppendLine('Review frontend/src/router/index.ts on upstream for new paths. Fork route ownership stays in frontend/src/hk/workspaceRoutes.ts.')
[void]$sb.AppendLine()
[void]$sb.AppendLine('## Migration numbers')
[void]$sb.AppendLine()
if ($migrationLines.Count -eq 0) { [void]$sb.AppendLine('No new overlapping migration prefixes.') }
else { foreach ($line in $migrationLines) { [void]$sb.AppendLine("- $line") } }
[void]$sb.AppendLine()
[void]$sb.AppendLine('## Largest remaining fork diffs')
[void]$sb.AppendLine()
foreach ($item in $hotspots) {
  [void]$sb.AppendLine("- $($item.Churn) (+$($item.Added)/-$($item.Deleted)) ``$($item.Path)``")
}
[void]$sb.AppendLine()
[void]$sb.AppendLine('## Checks after resolving conflicts')
[void]$sb.AppendLine()
[void]$sb.AppendLine('```')
[void]$sb.AppendLine('cd backend && go generate ./ent')
[void]$sb.AppendLine('cd backend && go generate ./cmd/server')
[void]$sb.AppendLine('cd backend && go test -tags=unit ./...')
[void]$sb.AppendLine('cd frontend && pnpm typecheck && pnpm lint:check && pnpm test:run && pnpm build')
[void]$sb.AppendLine('```')
[void]$sb.AppendLine()
[void]$sb.AppendLine('This script does not commit or push.')

[System.IO.File]::WriteAllText($reportPath, $sb.ToString())
Write-Host "Report: $reportPath"
Write-Host ("Predicted conflicts: {0}; auto-delete: {1}; ent: {2}; manual: {3}" -f $conflictPaths.Count, $autoRemoved.Count, $entConflicts.Count, $manual.Count)
if ($manual.Count -gt 0) {
  Write-Host 'Still needs a hand merge:'
  $manual | ForEach-Object { Write-Host "  $_" }
}
