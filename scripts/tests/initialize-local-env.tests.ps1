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

$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) "initialize-local-env.ps1"
Assert-True -Condition (Test-Path -LiteralPath $scriptPath) -Message "initialize-local-env.ps1 must exist"

$tempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
$testDirectory = Join-Path $tempRoot ("new-api-env-test-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $testDirectory | Out-Null

try {
  $examplePath = Join-Path $testDirectory ".env.example"
  $outputPath = Join-Path $testDirectory ".env"
  @"
POSTGRES_DB=new_api
POSTGRES_USER=new_api
POSTGRES_PASSWORD=SET_BY_INITIALIZE_LOCAL_ENV
SESSION_SECRET=SET_BY_INITIALIZE_LOCAL_ENV
"@ | Set-Content -LiteralPath $examplePath -Encoding utf8NoBOM

  $reportedPath = & $scriptPath -ExamplePath $examplePath -OutputPath $outputPath
  Assert-True -Condition (Test-Path -LiteralPath $outputPath) -Message "the generated .env file must exist"
  Assert-True -Condition ([System.IO.Path]::GetFullPath([string]$reportedPath) -eq [System.IO.Path]::GetFullPath($outputPath)) -Message "the script must report only the generated file path"

  $values = @{}
  foreach ($line in Get-Content -LiteralPath $outputPath) {
    if ($line -match "^(?<key>[^=]+)=(?<value>.*)$") {
      $values[$Matches.key] = $Matches.value
    }
  }

  Assert-True -Condition ($values.POSTGRES_PASSWORD -match "^[0-9a-f]{64}$") -Message "the PostgreSQL password must be 32 random bytes encoded as lowercase hex"
  Assert-True -Condition ($values.SESSION_SECRET -match "^[0-9a-f]{64}$") -Message "the session secret must be 32 random bytes encoded as lowercase hex"
  Assert-True -Condition ($values.POSTGRES_PASSWORD -ne $values.SESSION_SECRET) -Message "the two generated secrets must be independent"
  Assert-True -Condition (-not ((Get-Content -Raw -LiteralPath $outputPath).Contains("SET_BY_INITIALIZE_LOCAL_ENV"))) -Message "no sentinel may remain in the generated file"

  $overwritten = $false
  try {
    & $scriptPath -ExamplePath $examplePath -OutputPath $outputPath | Out-Null
    $overwritten = $true
  } catch {
    $overwritten = $false
  }
  Assert-True -Condition (-not $overwritten) -Message "the script must refuse to overwrite an existing .env file"

  "PASS initialize-local-env behavior"
} finally {
  $resolvedTestDirectory = [System.IO.Path]::GetFullPath($testDirectory)
  if (-not $resolvedTestDirectory.StartsWith($tempRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "refusing to remove a directory outside the system temp root"
  }
  Remove-Item -LiteralPath $resolvedTestDirectory -Recurse -Force
}
