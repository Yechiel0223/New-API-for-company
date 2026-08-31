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

$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) "test-seedance-text-video.ps1"
Assert-True -Condition (Test-Path -LiteralPath $scriptPath) -Message "test-seedance-text-video.ps1 must exist"

$previousKey = $env:NEW_API_KEY
$env:NEW_API_KEY = "test-secret-must-not-appear"

try {
  $output = & $scriptPath `
    -DryRun `
    -BaseUrl "http://127.0.0.1:3000/" `
    -Resolution "480p" `
    -Duration 5 `
    -Prompt "一架纸飞机平稳飞过纯白背景"

  $json = ($output | Out-String).Trim() | ConvertFrom-Json
  Assert-True -Condition ($json.mode -eq "dry-run") -Message "dry run must identify its mode"
  Assert-True -Condition ($json.method -eq "POST") -Message "dry run must use POST"
  Assert-True -Condition ($json.endpoint -eq "http://127.0.0.1:3000/doubao/api/v3/contents/generations/tasks") -Message "dry run endpoint is incorrect"
  Assert-True -Condition ($json.request.model -eq "doubao-seedance-2-5-260628") -Message "dry run model is incorrect"
  Assert-True -Condition ($json.request.resolution -eq "480p") -Message "dry run resolution is incorrect"
  Assert-True -Condition ([int]$json.request.duration -eq 5) -Message "dry run duration is incorrect"
  Assert-True -Condition (@($json.request.content).Count -eq 1) -Message "dry run must contain one text input"
  Assert-True -Condition ($json.request.content[0].type -eq "text") -Message "dry run content type is incorrect"
  Assert-True -Condition ($json.request.content[0].text -eq "一架纸飞机平稳飞过纯白背景") -Message "dry run prompt is incorrect"
  Assert-True -Condition (-not (($output | Out-String) -match "test-secret-must-not-appear")) -Message "dry run output must not reveal the API key"

  "PASS seedance dry-run request"

  $invalidResolutionRejected = $false
  try {
    & $scriptPath -DryRun -Resolution "4k" | Out-Null
  } catch {
    $invalidResolutionRejected = $_.Exception.Message -eq "Resolution must be 480p, 720p, or 1080p"
  }
  Assert-True -Condition $invalidResolutionRejected -Message "Seedance 2.5 must reject unsupported 4k requests"

  foreach ($invalidDuration in @(3, 31)) {
    $invalidDurationRejected = $false
    try {
      & $scriptPath -DryRun -Duration $invalidDuration | Out-Null
    } catch {
      $invalidDurationRejected = $_.Exception.Message -eq "Duration must be between 4 and 30 seconds"
    }
    Assert-True -Condition $invalidDurationRejected -Message "Seedance 2.5 must reject duration $invalidDuration"
  }

  "PASS seedance official parameter boundaries"

  $env:NEW_API_KEY = $null
  $missingKeyRejected = $false
  try {
    & $scriptPath -BaseUrl "http://127.0.0.1:1" -TimeoutSeconds 1 | Out-Null
  } catch {
    $missingKeyRejected = $_.Exception.Message -eq "NEW_API_KEY must be set for live mode"
  }
  Assert-True -Condition $missingKeyRejected -Message "live mode must reject a missing API key before making a request"
  $env:NEW_API_KEY = "test-secret-must-not-appear"

  "PASS seedance missing-key validation"

  $tempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
  $testDirectory = Join-Path $tempRoot ("new-api-seedance-test-" + [guid]::NewGuid().ToString("N"))
  New-Item -ItemType Directory -Path $testDirectory | Out-Null

  $portProbe = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
  $portProbe.Start()
  $port = ([System.Net.IPEndPoint]$portProbe.LocalEndpoint).Port
  $portProbe.Stop()

  $fixturePath = Join-Path $testDirectory "fixture.py"
  $readyPath = Join-Path $testDirectory "ready"
  $requestBodyPath = Join-Path $testDirectory "request.json"
  $requestLogPath = Join-Path $testDirectory "requests.log"
  $stdoutPath = Join-Path $testDirectory "fixture.stdout"
  $stderrPath = Join-Path $testDirectory "fixture.stderr"
  @'
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

port = int(sys.argv[1])
ready_path = Path(sys.argv[2])
request_body_path = Path(sys.argv[3])
request_log_path = Path(sys.argv[4])

class Handler(BaseHTTPRequestHandler):
    get_count = 0

    def send_json(self, payload, status=200):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        with request_log_path.open("a", encoding="utf-8") as request_log:
            request_log.write("POST\n")
        length = int(self.headers.get("Content-Length", "0"))
        request_body_path.write_bytes(self.rfile.read(length))
        self.send_json({"id": "task_test_123", "status": "queued"})

    def do_GET(self):
        with request_log_path.open("a", encoding="utf-8") as request_log:
            request_log.write("GET\n")
        Handler.get_count += 1
        if Handler.get_count == 1:
            self.send_json({"error": {"message": "temporary database outage"}}, 500)
            return
        self.send_json({
            "id": "task_test_123",
            "status": "succeeded",
            "usage": {"completion_tokens": 108000, "total_tokens": 108000},
            "content": {
                "video_url": "https://media.example/signed?token=do-not-store",
                "resolution": "720p",
            },
        })

    def log_message(self, format, *args):
        pass

server = HTTPServer(("127.0.0.1", port), Handler)
ready_path.write_text("ready", encoding="utf-8")
for _ in range(3):
    server.handle_request()
server.server_close()
'@ | Set-Content -LiteralPath $fixturePath -Encoding utf8

  $serverProcess = Start-Process `
    -FilePath "python.exe" `
    -ArgumentList @($fixturePath, [string]$port, $readyPath, $requestBodyPath, $requestLogPath) `
    -RedirectStandardOutput $stdoutPath `
    -RedirectStandardError $stderrPath `
    -WindowStyle Hidden `
    -PassThru

  try {
    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    do {
      if (Test-Path -LiteralPath $readyPath -PathType Leaf) {
        break
      }
      if ($serverProcess.HasExited) {
        $fixtureError = Get-Content -Raw -LiteralPath $stderrPath -ErrorAction SilentlyContinue
        throw "local HTTP fixture failed to start: $fixtureError"
      }
      Start-Sleep -Milliseconds 50
    } while ([DateTime]::UtcNow -lt $deadline)
    Assert-True -Condition (Test-Path -LiteralPath $readyPath -PathType Leaf) -Message "local HTTP fixture did not become ready"

    $liveOutput = & $scriptPath `
      -BaseUrl "http://127.0.0.1:$port" `
      -Resolution "720p" `
      -Duration 5 `
      -Prompt "一架纸飞机平稳飞过纯白背景" `
      -PollIntervalSeconds 0 `
      -TimeoutSeconds 10 `
      -OutputDirectory $testDirectory

    $liveJson = ($liveOutput | Out-String).Trim() | ConvertFrom-Json
    Assert-True -Condition ($liveJson.status -eq "succeeded") -Message "live mode must return the terminal status"
    Assert-True -Condition ($liveJson.task_id -eq "task_test_123") -Message "live mode must return the public task id"
    Assert-True -Condition ([int]$liveJson.completion_tokens -eq 108000) -Message "live mode must return actual completion tokens"
    Assert-True -Condition ($liveJson.video_url_present -eq $true) -Message "live mode must record that a video URL exists"
    Assert-True -Condition ([int]$liveJson.query_retry_count -eq 1) -Message "live mode must report one transient query retry"
    Assert-True -Condition (Test-Path -LiteralPath $liveJson.evidence_path -PathType Leaf) -Message "live mode must save a sanitized evidence file"

    $evidence = Get-Content -Raw -LiteralPath $liveJson.evidence_path
    Assert-True -Condition (-not ($evidence -match "test-secret-must-not-appear")) -Message "evidence must not reveal the API key"
    Assert-True -Condition (-not ($evidence -match "media\.example|do-not-store")) -Message "evidence must not save the signed video URL"
    Assert-True -Condition (($evidence | ConvertFrom-Json).completion_tokens -eq 108000) -Message "evidence must retain actual completion tokens"
    $requestSequence = @((Get-Content -LiteralPath $requestLogPath)) -join ","
    Assert-True -Condition ($requestSequence -eq "POST,GET,GET") -Message "a transient query 5xx must retry GET without submitting another POST"

    "PASS seedance submit, transient query retry, poll, and sanitized evidence"
  } finally {
    if (-not $serverProcess.HasExited) {
      Stop-Process -Id $serverProcess.Id -Force
    }

    $resolvedTestDirectory = [System.IO.Path]::GetFullPath($testDirectory)
    if (-not $resolvedTestDirectory.StartsWith($tempRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
      throw "refusing to remove a directory outside the system temp root"
    }
    Remove-Item -LiteralPath $resolvedTestDirectory -Recurse -Force
  }
} finally {
  $env:NEW_API_KEY = $previousKey
}
