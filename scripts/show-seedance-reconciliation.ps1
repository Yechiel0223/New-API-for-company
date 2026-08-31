param(
  [Parameter(Mandatory)]
  [string]$TaskId,

  [Parameter(Mandatory)]
  [string]$ComposePath,

  [Parameter(Mandatory)]
  [string]$EnvPath
)

$ErrorActionPreference = "Stop"

if ($TaskId -notmatch '^task_[A-Za-z0-9_-]+$') {
  throw "Task id format is invalid"
}

$composeFullPath = [System.IO.Path]::GetFullPath($ComposePath)
$envFullPath = [System.IO.Path]::GetFullPath($EnvPath)
if (-not (Test-Path -LiteralPath $composeFullPath -PathType Leaf)) {
  throw "Compose file does not exist"
}
if (-not (Test-Path -LiteralPath $envFullPath -PathType Leaf)) {
  throw "Environment file does not exist"
}

$dockerCommand = Get-Command docker.exe -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Source
if (-not $dockerCommand) {
  $dockerDesktopCommand = "C:\Program Files\Docker\Docker\resources\bin\docker.exe"
  if (Test-Path -LiteralPath $dockerDesktopCommand -PathType Leaf) {
    $dockerCommand = $dockerDesktopCommand
  }
}
if (-not $dockerCommand) {
  throw "Docker CLI does not exist"
}

$querySql = @"
select json_build_object(
  'task_id',t.task_id,
  'status',t.status,
  'action',t.action,
  'model',coalesce(t.properties->>'origin_model_name',''),
  'channel_id',t.channel_id,
  'token_id',coalesce(nullif(t.private_data->>'token_id','')::bigint,0),
  'token_name',coalesce(tok.name,''),
  'quota',t.quota,
  'estimated_quota',coalesce(nullif(t.private_data->'billing_context'->'tiered_snapshot'->>'estimated_quota_after_group','')::numeric,0),
  'usage_tokens',coalesce(nullif(t.private_data->'billing_context'->'tiered_snapshot'->'usage_facts'->>'tokens','')::numeric,0),
  'resolution',coalesce(t.private_data->'billing_context'->'tiered_snapshot'->'usage_facts'->>'resolution',''),
  'video_input',coalesce(t.private_data->'billing_context'->'tiered_snapshot'->'usage_facts'->>'video_input',''),
  'token_remain_quota',coalesce(tok.remain_quota,0),
  'token_used_quota',coalesce(tok.used_quota,0)
)::text
from tasks t
left join tokens tok on tok.id=coalesce(nullif(t.private_data->>'token_id','')::bigint,0)
where t.task_id='$TaskId'
limit 1;
"@
$row = & $dockerCommand compose --env-file $envFullPath -f $composeFullPath exec -T postgres `
  psql -U new_api -d new_api -Atc $querySql 2>$null
if ($LASTEXITCODE -ne 0) {
  throw "Database query failed"
}
$rowJson = ($row | Out-String).Trim()
if ([string]::IsNullOrWhiteSpace($rowJson)) {
  throw "Task not found"
}

try {
  $data = $rowJson | ConvertFrom-Json
} catch {
  throw "Database returned invalid reconciliation data"
}

$quotaPerCny = [decimal]500000
$quota = [decimal]$data.quota
$estimatedQuota = [decimal]$data.estimated_quota
$usageTokens = [decimal]$data.usage_tokens
$resolution = [string]$data.resolution
$chargedCny = [decimal]::Round($quota / $quotaPerCny, 6, [MidpointRounding]::AwayFromZero)
$estimatedCny = [decimal]::Round($estimatedQuota / $quotaPerCny, 6, [MidpointRounding]::AwayFromZero)

$unitPrice = [decimal]0
if ($resolution -eq "1080p") {
  $unitPrice = [decimal]55.44
} elseif (@("480p", "720p") -contains $resolution) {
  $unitPrice = [decimal]70
}

$recomputedCny = [decimal]0
if ($unitPrice -gt 0 -and $usageTokens -gt 0) {
  $recomputedCny = [decimal]::Round($usageTokens * $unitPrice / 1000000, 6, [MidpointRounding]::AwayFromZero)
}
$reconciliationDelta = [decimal]::Round($chargedCny - $recomputedCny, 6, [MidpointRounding]::AwayFromZero)

[ordered]@{
  task_id = [string]$data.task_id
  status = [string]$data.status
  action = [string]$data.action
  model = [string]$data.model
  channel_id = [int]$data.channel_id
  token_id = [int]$data.token_id
  token_name = [string]$data.token_name
  quota = [int64]$data.quota
  charged_cny = $chargedCny
  estimated_quota = [int64]$data.estimated_quota
  estimated_cny = $estimatedCny
  usage_tokens = [int64]$data.usage_tokens
  resolution = $resolution
  video_input = [string]$data.video_input
  unit_price_cny_per_million = $unitPrice
  recomputed_cny = $recomputedCny
  reconciliation_delta_cny = $reconciliationDelta
  billing_matches = [math]::Abs([double]$reconciliationDelta) -le 0.000002
  token_remain_cny = [decimal]::Round(([decimal]$data.token_remain_quota) / $quotaPerCny, 6, [MidpointRounding]::AwayFromZero)
  token_used_cny = [decimal]::Round(([decimal]$data.token_used_quota) / $quotaPerCny, 6, [MidpointRounding]::AwayFromZero)
} | ConvertTo-Json -Depth 10
