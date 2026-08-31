$ErrorActionPreference = "Stop"

function Assert-True {
  param(
    [Parameter(Mandatory)]
    [bool]$Condition,

    [Parameter(Mandatory)]
    [string]$Message
  )

  if (-not $Condition) {
    throw $Message
  }
}

$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) "show-seedance-reconciliation.ps1"
Assert-True -Condition (Test-Path -LiteralPath $scriptPath) -Message "show-seedance-reconciliation.ps1 must exist"

$repoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$composePath = Join-Path $repoRoot "deploy\compose.yaml"
$envPath = Join-Path $repoRoot "deploy\.env"
$notFound = $false

try {
  & $scriptPath `
    -TaskId "task_missing_for_reconciliation_test" `
    -ComposePath $composePath `
    -EnvPath $envPath | Out-Null
} catch {
  $notFound = $_.Exception.Message -eq "Task not found"
}

Assert-True -Condition $notFound -Message "reconciliation must report an unknown task without exposing database details"
"PASS seedance reconciliation unknown-task behavior"

$dockerCommand = Get-Command docker.exe -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Source
if (-not $dockerCommand) {
  $dockerCommand = "C:\Program Files\Docker\Docker\resources\bin\docker.exe"
}
Assert-True -Condition (Test-Path -LiteralPath $dockerCommand -PathType Leaf) -Message "Docker CLI must exist for reconciliation fixture"

$fixtureSuffix = [guid]::NewGuid().ToString("N")
$fixtureTaskId = "task_reconciliation_$fixtureSuffix"
$fixture1080TaskId = "task_reconciliation_1080_$fixtureSuffix"
$fixtureKey = "reconciliation-fixture-$fixtureSuffix"
$fixtureTokenName = "reconciliation-fixture"
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$insertTokenSql = @"
insert into tokens (user_id,key,status,name,created_time,accessed_time,expired_time,remain_quota,unlimited_quota,model_limits_enabled,model_limits,used_quota,cross_group_retry,auto_groups)
values (1,'$fixtureKey',1,'$fixtureTokenName',$timestamp,$timestamp,-1,100000000,false,true,'doubao-seedance-2-5-260628',0,false,'') returning id;
"@
$insertTokenOutput = @(& $dockerCommand compose --env-file $envPath -f $composePath exec -T postgres `
  psql -U new_api -d new_api -Atc $insertTokenSql 2>$null)
$insertTokenExitCode = $LASTEXITCODE
$fixtureTokenId = @($insertTokenOutput | ForEach-Object { ([string]$_).Trim() } | Where-Object { $_ -match '^\d+$' } | Select-Object -First 1)
Assert-True -Condition ($insertTokenExitCode -eq 0 -and $fixtureTokenId.Count -eq 1) -Message "reconciliation token fixture must be created"
$fixtureTokenId = $fixtureTokenId[0]

try {
  $insertTaskSql = @"
insert into tasks (created_at,updated_at,task_id,platform,user_id,channel_id,quota,action,status,submit_time,start_time,finish_time,progress,properties,private_data,data)
values (
  $timestamp,$timestamp,'$fixtureTaskId','45',1,1,3815000,'text_to_video','SUCCESS',$timestamp,$timestamp,$timestamp,'100%',
  json_build_object('origin_model_name','doubao-seedance-2-5-260628'),
  json_build_object(
    'token_id',$fixtureTokenId,
    'billing_context',json_build_object(
      'tiered_snapshot',json_build_object(
        'estimated_quota_after_group',3780000,
        'usage_facts',json_build_object('tokens',108000,'resolution','720p','video_input','none')
      )
    )
  ),
  json_build_object('usage',json_build_object('completion_tokens',109000,'total_tokens',109000))
);
"@
  $null = & $dockerCommand compose --env-file $envPath -f $composePath exec -T postgres `
    psql -U new_api -d new_api -Atc $insertTaskSql 2>$null
  Assert-True -Condition ($LASTEXITCODE -eq 0) -Message "reconciliation task fixture must be created"

  $insert1080TaskSql = @"
insert into tasks (created_at,updated_at,task_id,platform,user_id,channel_id,quota,action,status,submit_time,start_time,finish_time,progress,properties,private_data,data)
values (
  $timestamp,$timestamp,'$fixture1080TaskId','45',1,1,6735960,'text_to_video','SUCCESS',$timestamp,$timestamp,$timestamp,'100%',
  json_build_object('origin_model_name','doubao-seedance-2-5-260628'),
  json_build_object(
    'token_id',$fixtureTokenId,
    'billing_context',json_build_object(
      'tiered_snapshot',json_build_object(
        'estimated_quota_after_group',6735960,
        'usage_facts',json_build_object('tokens',243000,'resolution','1080p','video_input','none')
      )
    )
  ),
  json_build_object()
);
"@
  $null = & $dockerCommand compose --env-file $envPath -f $composePath exec -T postgres `
    psql -U new_api -d new_api -Atc $insert1080TaskSql 2>$null
  Assert-True -Condition ($LASTEXITCODE -eq 0) -Message "1080p reconciliation task fixture must be created"

  $output = & $scriptPath -TaskId $fixtureTaskId -ComposePath $composePath -EnvPath $envPath
  $json = ($output | Out-String).Trim() | ConvertFrom-Json
  Assert-True -Condition ($json.task_id -eq $fixtureTaskId) -Message "reconciliation must return the public task id"
  Assert-True -Condition ($json.status -eq "SUCCESS") -Message "reconciliation must return the task status"
  Assert-True -Condition ($json.token_name -eq $fixtureTokenName) -Message "reconciliation must return the non-secret token name"
  Assert-True -Condition ([int64]$json.quota -eq 3815000) -Message "reconciliation must return final raw quota"
  Assert-True -Condition ([decimal]$json.charged_cny -eq 7.63) -Message "reconciliation must convert quota to CNY"
  Assert-True -Condition ([int64]$json.estimated_quota -eq 3780000) -Message "reconciliation must retain the submission estimate"
  Assert-True -Condition ([decimal]$json.estimated_cny -eq 7.56) -Message "reconciliation must convert the submission estimate to CNY"
  Assert-True -Condition ([int64]$json.estimated_usage_tokens -eq 108000) -Message "reconciliation must expose estimated usage separately"
  Assert-True -Condition ([int64]$json.usage_tokens -eq 109000) -Message "reconciliation must prefer actual task usage over estimated snapshot usage"
  Assert-True -Condition ($json.usage_source -eq "task.data.usage.completion_tokens") -Message "reconciliation must identify the actual usage source"
  Assert-True -Condition ($json.resolution -eq "720p") -Message "reconciliation must return the billed resolution"
  Assert-True -Condition ([decimal]$json.unit_price_cny_per_million -eq 70) -Message "reconciliation must select the 720p unit price"
  Assert-True -Condition ([decimal]$json.recomputed_cny -eq 7.63) -Message "reconciliation must recompute CNY from actual usage"
  Assert-True -Condition ([decimal]$json.reconciliation_delta_cny -eq 0) -Message "reconciliation must report zero delta for matching billing"
  Assert-True -Condition (-not (($output | Out-String) -match [regex]::Escape($fixtureKey))) -Message "reconciliation output must not reveal token keys"
  Assert-True -Condition (-not (($output | Out-String) -match "private_data|result_url|upstream_task_id")) -Message "reconciliation output must not reveal private task fields"

  "PASS seedance reconciliation CNY and usage behavior"

  $output1080 = & $scriptPath -TaskId $fixture1080TaskId -ComposePath $composePath -EnvPath $envPath
  $json1080 = ($output1080 | Out-String).Trim() | ConvertFrom-Json
  Assert-True -Condition ([int64]$json1080.quota -eq 6735960) -Message "1080p reconciliation must return final raw quota"
  Assert-True -Condition ([decimal]$json1080.charged_cny -eq 13.47192) -Message "1080p reconciliation must convert quota to CNY"
  Assert-True -Condition ([int64]$json1080.usage_tokens -eq 243000) -Message "1080p reconciliation must return actual usage tokens"
  Assert-True -Condition ($json1080.resolution -eq "1080p") -Message "1080p reconciliation must return its billed resolution"
  Assert-True -Condition ([decimal]$json1080.unit_price_cny_per_million -eq 55.44) -Message "1080p reconciliation must use the current promotional unit price"
  Assert-True -Condition ([decimal]$json1080.recomputed_cny -eq 13.47192) -Message "1080p reconciliation must recompute the promotional CNY charge"
  Assert-True -Condition ([decimal]$json1080.reconciliation_delta_cny -eq 0) -Message "1080p reconciliation must report zero delta for matching billing"

  "PASS seedance 1080p promotional reconciliation behavior"
} finally {
  $deleteTaskSql = "delete from tasks where task_id in ('$fixtureTaskId','$fixture1080TaskId');"
  $deleteTokenSql = "delete from tokens where id=$fixtureTokenId;"
  $null = & $dockerCommand compose --env-file $envPath -f $composePath exec -T postgres `
    psql -U new_api -d new_api -Atc $deleteTaskSql 2>$null
  $null = & $dockerCommand compose --env-file $envPath -f $composePath exec -T postgres `
    psql -U new_api -d new_api -Atc $deleteTokenSql 2>$null
}
