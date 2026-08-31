param(
  [string]$BaseUrl = "http://localhost:3000",

  [string]$Resolution = "720p",

  [int]$Duration = 5,

  [string]$Prompt = "一架纸飞机平稳飞过纯白背景",

  [ValidateRange(0, 300)]
  [int]$PollIntervalSeconds = 5,

  [ValidateRange(1, 3600)]
  [int]$TimeoutSeconds = 1800,

  [string]$OutputDirectory = (Join-Path (Split-Path -Parent $PSScriptRoot) "artifacts\poc"),

  [switch]$DryRun
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Prompt)) {
  throw "Prompt must not be empty"
}
if (@("480p", "720p", "1080p") -notcontains $Resolution) {
  throw "Resolution must be 480p, 720p, or 1080p"
}
if ($Duration -lt 4 -or $Duration -gt 30) {
  throw "Duration must be between 4 and 30 seconds"
}

$endpoint = $BaseUrl.TrimEnd("/") + "/doubao/api/v3/contents/generations/tasks"
$request = [ordered]@{
  model = "doubao-seedance-2-5-260628"
  content = @(
    [ordered]@{
      type = "text"
      text = $Prompt
    }
  )
  resolution = $Resolution
  duration = $Duration
}

if (-not $DryRun) {
  if ([string]::IsNullOrWhiteSpace($env:NEW_API_KEY)) {
    throw "NEW_API_KEY must be set for live mode"
  }

  $headers = @{
    Authorization = "Bearer $($env:NEW_API_KEY)"
    Accept = "application/json"
  }
  $requestJson = $request | ConvertTo-Json -Depth 10
  $startedAt = [DateTime]::UtcNow

  try {
    $submitted = Invoke-RestMethod `
      -Uri $endpoint `
      -Method Post `
      -Headers $headers `
      -ContentType "application/json; charset=utf-8" `
      -Body $requestJson
  } catch {
    throw "Seedance submit failed: $($_.Exception.Message)"
  }

  $taskId = [string]$submitted.id
  if ([string]::IsNullOrWhiteSpace($taskId)) {
    throw "Seedance submit response did not contain a task id"
  }

  $queryEndpoint = $endpoint + "/" + [uri]::EscapeDataString($taskId)
  $terminalStatuses = @("succeeded", "failed", "expired", "cancelled")
  $task = $null

  while ($true) {
    if (([DateTime]::UtcNow - $startedAt).TotalSeconds -ge $TimeoutSeconds) {
      throw "Seedance task polling timed out"
    }
    if ($PollIntervalSeconds -gt 0) {
      Start-Sleep -Seconds $PollIntervalSeconds
    }

    try {
      $task = Invoke-RestMethod -Uri $queryEndpoint -Method Get -Headers $headers
    } catch {
      throw "Seedance query failed: $($_.Exception.Message)"
    }

    $status = ([string]$task.status).ToLowerInvariant()
    if ($terminalStatuses -contains $status) {
      break
    }
  }

  $completionTokens = 0
  $totalTokens = 0
  if ($null -ne $task.usage) {
    $completionTokens = [int64]$task.usage.completion_tokens
    $totalTokens = [int64]$task.usage.total_tokens
  }
  $videoUrlPresent = $null -ne $task.content -and -not [string]::IsNullOrWhiteSpace([string]$task.content.video_url)
  $finishedAt = [DateTime]::UtcNow
  $evidence = [ordered]@{
    started_at = $startedAt.ToString("o")
    finished_at = $finishedAt.ToString("o")
    task_id = $taskId
    model = $request.model
    resolution = $Resolution
    duration = $Duration
    prompt = $Prompt
    status = $status
    completion_tokens = $completionTokens
    total_tokens = $totalTokens
    video_url_present = $videoUrlPresent
  }

  $outputFullPath = [System.IO.Path]::GetFullPath($OutputDirectory)
  New-Item -ItemType Directory -Path $outputFullPath -Force | Out-Null
  $safeTaskId = $taskId -replace '[^A-Za-z0-9_-]', '_'
  $evidencePath = Join-Path $outputFullPath ("seedance-$safeTaskId.json")
  $evidence | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $evidencePath -Encoding utf8

  $result = [ordered]@{}
  foreach ($entry in $evidence.GetEnumerator()) {
    $result[$entry.Key] = $entry.Value
  }
  $result.evidence_path = $evidencePath
  $result | ConvertTo-Json -Depth 10
  return
}

[ordered]@{
  mode = "dry-run"
  method = "POST"
  endpoint = $endpoint
  request = $request
} | ConvertTo-Json -Depth 10
