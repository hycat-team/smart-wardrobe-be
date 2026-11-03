# Feature 032 - local functional checks and CPU latency sampling for the Ollama trial.
#
# Runbook: docs/operations/ollama-trial-runbook.md sections A3 and A5.
# Contract: specs/032-ollama-lightsail-trial/contracts/ollama-openai-subset.md
#
# This script is READ-ONLY with respect to the host: it never installs Ollama, never
# pulls a model and never edits configuration. If the Ollama CLI or server is missing it
# stops with a clear message; see runbook A1 for the manual install/pull step.
#
# What it verifies
#   1. Preflight: ollama CLI present, server reachable, model available.
#   2. Non-stream /v1/chat/completions returns content.
#   3. Streaming SSE reaches "data: [DONE]" and emits chunks.
#   4. Structured output (response_format json_object) parses and has the expected keys.
#   5. Thinking is off: no reasoning_content / thinking, reasoning_tokens = 0.
#   6. CPU latency samples (TTFT + total) with bounded concurrency, raw CSV + summary.
#
# Default output goes to ./docs/temp/ollama-trial (gitignored). Never point it at a
# tracked directory: raw samples are evidence, not source.

[CmdletBinding()]
param(
    [string]$BaseUrl = "http://127.0.0.1:11434",
    [string]$Model = "qwen2.5:3b-instruct-q4_K_M",
    [string]$ReasoningEffort = "",
    [int]$MaxTokens = 64,
    [int]$Samples = 5,
    [int]$Concurrency = 2,
    [string]$Capability = "chat_text",
    [string]$Prompt = "Explain in two short sentences why regular sleep matters.",
    [string]$OutDir = "./docs/temp/ollama-trial",
    [switch]$SkipFunctional,
    [switch]$SkipLatency,
    [switch]$AllowGpu
)

$ErrorActionPreference = "Stop"

function Write-Step([string]$Message) {
    Write-Host "[032-local-test] $Message"
}

function Fail([string]$Message, [int]$Code = 1) {
    Write-Host "[032-local-test] FAIL: $Message" -ForegroundColor Red
    exit $Code
}

function Get-ChatUri { param([string]$Url) return ($Url.TrimEnd('/') + "/v1/chat/completions") }

function New-ChatBody {
    param(
        [string]$UserPrompt,
        [bool]$Stream = $false,
        [bool]$JsonMode = $false,
        [int]$Tokens = 64
    )
    $body = [ordered]@{
        model            = $Model
        messages         = @(
            @{ role = "system"; content = "Answer briefly." },
            @{ role = "user"; content = $UserPrompt }
        )
        stream           = $Stream
        max_tokens       = $Tokens
        temperature      = 0
    }
    if ($ReasoningEffort -ne "") {
        $body["reasoning_effort"] = $ReasoningEffort
    }
    if ($JsonMode) {
        $body["response_format"] = @{ type = "json_object" }
    }
    return ($body | ConvertTo-Json -Depth 6)
}

function Get-MessageContent {
    param($Response)
    if ($null -eq $Response -or $null -eq $Response.choices -or $Response.choices.Count -eq 0) {
        throw "response has no choices"
    }
    $content = $Response.choices[0].message.content
    if ($null -eq $content) { return "" }
    return [string]$content
}

function ConvertFrom-ModelJson {
    param([string]$Text)
    $clean = $Text.Trim()
    if ($clean.StartsWith('```')) {
        $lines = $clean -split "`r?`n"
        if ($lines.Count -ge 3) {
            $clean = ($lines[1..($lines.Count - 2)] -join "`n").Trim()
        }
    }
    return ($clean | ConvertFrom-Json)
}

function Invoke-ChatNonStream {
    param([string]$Prompt, [bool]$JsonMode = $false, [int]$Tokens = 64)
    $uri = Get-ChatUri $BaseUrl
    $body = New-ChatBody -UserPrompt $Prompt -Stream $false -JsonMode $JsonMode -Tokens $Tokens
    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        $resp = Invoke-RestMethod -Method Post -Uri $uri -ContentType "application/json" -Body $body -TimeoutSec 180
    }
    catch {
        throw ("non-stream request failed: " + $_.Exception.Message)
    }
    $sw.Stop()
    return [pscustomobject]@{ Response = $resp; LatencyMs = $sw.ElapsedMilliseconds }
}

function Invoke-ChatStream {
    param([string]$Prompt, [int]$Tokens = 64)
    $uri = Get-ChatUri $BaseUrl
    $bodyText = New-ChatBody -UserPrompt $Prompt -Stream $true -JsonMode $false -Tokens $Tokens
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($bodyText)

    $request = [System.Net.HttpWebRequest]::Create($uri)
    $request.Method = "POST"
    $request.ContentType = "application/json"
    $request.Accept = "text/event-stream"
    $request.Timeout = 180000
    $request.ReadWriteTimeout = 180000
    $request.AllowReadStreamBuffering = $false
    $request.ContentLength = $bytes.Length

    $stream = $request.GetRequestStream()
    try {
        $stream.Write($bytes, 0, $bytes.Length)
    }
    finally {
        $stream.Close()
    }

    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    $ttft = $null
    $text = New-Object System.Text.StringBuilder
    $sawDone = $false
    $thinkingSeen = $false
    $sawChunk = $false
    $reasoningTokens = $null

    try {
        $response = $request.GetResponse()
    }
    catch {
        throw ("stream request failed: " + $_.Exception.Message)
    }

    $reader = $null
    $reader = New-Object System.IO.StreamReader($response.GetResponseStream())
    try {
        while ($true) {
            $line = $reader.ReadLine()
            if ($null -eq $line) { break }
            if (-not $line.StartsWith("data:")) { continue }

            $payload = $line.Substring(5).Trim()
            if ($payload -eq "[DONE]") { $sawDone = $true; break }
            if ($payload.Length -eq 0) { continue }

            try {
                $chunk = $payload | ConvertFrom-Json
            }
            catch {
                throw "malformed SSE chunk from provider"
            }

            if ($null -ne $chunk.choices -and $chunk.choices.Count -gt 0) {
                $delta = $chunk.choices[0].delta
                if ($null -ne $delta) {
                    if ($null -ne $delta.reasoning_content -and ([string]$delta.reasoning_content).Length -gt 0) {
                        $thinkingSeen = $true
                    }
                    if ($null -ne $delta.thinking -and ([string]$delta.thinking).Length -gt 0) {
                        $thinkingSeen = $true
                    }
                    if ($null -ne $delta.content -and ([string]$delta.content).Length -gt 0) {
                        $sawChunk = $true
                        if ($null -eq $ttft) { $ttft = $sw.ElapsedMilliseconds }
                        [void]$text.Append([string]$delta.content)
                    }
                }
            }

            if ($null -ne $chunk.usage -and $null -ne $chunk.usage.completion_tokens_details) {
                if ($null -ne $chunk.usage.completion_tokens_details.reasoning_tokens) {
                    $reasoningTokens = [int]$chunk.usage.completion_tokens_details.reasoning_tokens
                }
            }
        }
    }
    finally {
        $reader.Close()
        $response.Close()
        $sw.Stop()
    }

    return [pscustomobject]@{
        Text           = $text.ToString()
        SawChunk       = $sawChunk
        SawDone        = $sawDone
        ThinkingSeen   = $thinkingSeen
        TtftMs         = $ttft
        LatencyMs      = $sw.ElapsedMilliseconds
        ReasoningTokens = $reasoningTokens
    }
}

function Assert-NoThinking {
    param($NonStreamResponse, $StreamResult)
    $message = $NonStreamResponse.choices[0].message

    if ($null -ne $message.reasoning_content -and ([string]$message.reasoning_content).Length -gt 0) {
        throw "thinking is ON: non-stream response contains reasoning_content"
    }
    if ($null -ne $message.thinking -and ([string]$message.thinking).Length -gt 0) {
        throw "thinking is ON: non-stream response contains message.thinking"
    }
    if ($null -ne $NonStreamResponse.usage -and $null -ne $NonStreamResponse.usage.completion_tokens_details) {
        $rt = $NonStreamResponse.usage.completion_tokens_details.reasoning_tokens
        if ($null -ne $rt -and [int]$rt -gt 0) {
            throw "thinking is ON: usage completion_tokens_details.reasoning_tokens=$rt"
        }
    }
    if ($StreamResult.ThinkingSeen) {
        throw "thinking is ON: stream carried reasoning_content/thinking"
    }
    if ($null -ne $StreamResult.ReasoningTokens -and $StreamResult.ReasoningTokens -gt 0) {
        throw "thinking is ON: stream usage reasoning_tokens=$($StreamResult.ReasoningTokens)"
    }
}

# ------------------------------------------------------------------------------
# Preflight - never assume Ollama is installed
# ------------------------------------------------------------------------------
Write-Step "preflight"

$ollamaCmd = Get-Command ollama -ErrorAction SilentlyContinue
if ($null -eq $ollamaCmd) {
    Fail "Ollama CLI not found in PATH. Install Ollama natively and pull qwen2.5:3b-instruct-q4_K_M first (runbook A1, manual step). This script does not install or pull anything." 2
}

try {
    $version = (& ollama --version 2>&1 | Out-String).Trim()
}
catch {
    Fail ("ollama CLI is present but 'ollama --version' failed: " + $_.Exception.Message) 2
}
if ($LASTEXITCODE -ne 0) {
    Fail "ollama CLI is present but 'ollama --version' exited non-zero" 2
}
if ([string]::IsNullOrWhiteSpace($version)) {
    Fail "could not read 'ollama --version' output" 2
}
Write-Step "ollama: $version"

try {
    $serverVersion = Invoke-RestMethod -Method Get -Uri ($BaseUrl.TrimEnd('/') + "/api/version") -TimeoutSec 10
}
catch {
    Fail ("Ollama server is not reachable at $BaseUrl. Start 'ollama serve' or the Ollama app, then re-run. Detail: " + $_.Exception.Message) 2
}
Write-Step ("server: " + ($serverVersion | ConvertTo-Json -Compress))

$listed = (& ollama list 2>&1 | Out-String)
if ($listed -notmatch [regex]::Escape($Model)) {
    Fail "model '$Model' is not present. Run 'ollama pull $Model' manually (runbook A1). This script does not pull models." 2
}
$modelLine = ($listed -split "`r?`n" | Where-Object { $_ -match ("^" + [regex]::Escape($Model) + "\s") } | Select-Object -First 1)
Write-Step "model: $modelLine"

if (-not (Test-Path -LiteralPath $OutDir)) {
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
}

$summary = [ordered]@{
    timestampUtc     = (Get-Date).ToUniversalTime().ToString("o")
    baseUrl          = $BaseUrl
    model            = $Model
    capability       = $Capability
    ollamaVersion    = $version
    reasoningEffort  = $ReasoningEffort
    functionalPassed = $false
    latencySampled   = $false
    samples          = 0
    concurrency      = $Concurrency
    notes            = @()
}

# ------------------------------------------------------------------------------
# Functional checks
# ------------------------------------------------------------------------------
if (-not $SkipFunctional) {
    Write-Step "functional: non-stream"
    $plain = Invoke-ChatNonStream -Prompt "Say hello in one short sentence." -Tokens $MaxTokens
    $plainText = Get-MessageContent $plain.Response
    if ([string]::IsNullOrWhiteSpace($plainText)) { Fail "non-stream response content is empty" }
    Write-Step ("non-stream OK (" + $plain.LatencyMs + " ms): " + $plainText.Trim())

    Write-Step "functional: streaming"
    $streamResult = Invoke-ChatStream -Prompt "Count from one to five, one number per line." -Tokens $MaxTokens
    if (-not $streamResult.SawChunk) { Fail "stream produced no content chunk" }
    if (-not $streamResult.SawDone) { Fail "stream did not terminate with data: [DONE]" }
    if ($null -eq $streamResult.TtftMs) { Fail "stream TTFT was not measured" }
    Write-Step ("stream OK (ttft=" + $streamResult.TtftMs + " ms, total=" + $streamResult.LatencyMs + " ms)")

    Write-Step "functional: structured output"
    $jsonPrompt = 'Return only a JSON object with keys "ok" (boolean true) and "capability" (string "local-test"). No markdown.'
    $jsonResult = Invoke-ChatNonStream -Prompt $jsonPrompt -JsonMode $true -Tokens $MaxTokens
    $jsonText = Get-MessageContent $jsonResult.Response
    $parsed = $null
    try {
        $parsed = ConvertFrom-ModelJson $jsonText
    }
    catch {
        Fail ("structured output is not valid JSON: " + $_.Exception.Message)
    }
    if ($null -eq $parsed.ok -or $null -eq $parsed.capability) {
        Fail "structured output is missing required keys (ok, capability)"
    }
    if ($parsed.ok -ne $true) { Fail "structured output has ok != true" }
    Write-Step "structured output OK"

    Write-Step "functional: thinking off"
    try {
        Assert-NoThinking -NonStreamResponse $plain.Response -StreamResult $streamResult
    }
    catch {
        Fail $_.Exception.Message
    }
    Write-Step "thinking off OK"

    $summary.functionalPassed = $true
}

# ------------------------------------------------------------------------------
# CPU latency sampling
# ------------------------------------------------------------------------------
if (-not $SkipLatency) {
    $gpuSetting = $env:OLLAMA_NUM_GPU
    if (($null -eq $gpuSetting -or $gpuSetting -ne "0") -and -not $AllowGpu) {
        $summary.notes += "latency sampling skipped: OLLAMA_NUM_GPU is not '0'. Set `$env:OLLAMA_NUM_GPU='0', restart the Ollama server, then re-run. Use -AllowGpu only for a GPU reference measurement (never for SC-005)."
        Write-Step "latency: SKIPPED (OLLAMA_NUM_GPU is not '0'; see runbook A5)"
    }
    else {
        if ($gpuSetting -ne "0") {
            $summary.notes += "GPU reference measurement requested with -AllowGpu; these numbers must NOT replace SC-005."
            Write-Warning "Sampling with allow-gpu: results are a functional reference only, not SC-005 evidence."
        }
        if ($Concurrency -lt 1) { Fail "-Concurrency must be >= 1" }
        if ($Samples -lt 1) { Fail "-Samples must be >= 1" }

        Write-Step "latency: $Samples samples, concurrency $Concurrency"
        $worker = {
            param($Url, $ModelName, $Effort, $Prompt, $Tokens)
            $uri = $Url.TrimEnd('/') + "/v1/chat/completions"
            $body = [ordered]@{
                model            = $ModelName
                messages         = @(
                    @{ role = "system"; content = "Answer briefly." },
                    @{ role = "user"; content = $Prompt }
                )
                stream           = $true
                max_tokens       = $Tokens
                temperature      = 0
            }
            if ($Effort -ne "") { $body["reasoning_effort"] = $Effort }
            $bodyText = $body | ConvertTo-Json -Depth 6
            $bytes = [System.Text.Encoding]::UTF8.GetBytes($bodyText)

            $req = [System.Net.HttpWebRequest]::Create($uri)
            $req.Method = "POST"
            $req.ContentType = "application/json"
            $req.Accept = "text/event-stream"
            $req.Timeout = 180000
            $req.ReadWriteTimeout = 180000
            $req.AllowReadStreamBuffering = $false
            $req.ContentLength = $bytes.Length
            $s = $req.GetRequestStream()
            $s.Write($bytes, 0, $bytes.Length)
            $s.Close()

            $sw = [System.Diagnostics.Stopwatch]::StartNew()
            $ttft = $null
            $chars = 0
            $outcome = "success"
            $errorText = ""
            try {
                $resp = $req.GetResponse()
                $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
                try {
                    while ($true) {
                        $line = $reader.ReadLine()
                        if ($null -eq $line) { break }
                        if (-not $line.StartsWith("data:")) { continue }
                        $payload = $line.Substring(5).Trim()
                        if ($payload -eq "[DONE]") { break }
                        if ($payload.Length -eq 0) { continue }
                        try { $chunk = $payload | ConvertFrom-Json } catch { $outcome = "malformed"; break }
                        if ($null -ne $chunk.choices -and $chunk.choices.Count -gt 0) {
                            $delta = $chunk.choices[0].delta
                            if ($null -ne $delta -and $null -ne $delta.content -and ([string]$delta.content).Length -gt 0) {
                                if ($null -eq $ttft) { $ttft = $sw.ElapsedMilliseconds }
                                $chars += ([string]$delta.content).Length
                            }
                        }
                    }
                }
                finally {
                    $reader.Close()
                    $resp.Close()
                }
            }
            catch {
                $outcome = "error"
                $errorText = $_.Exception.Message
            }
            finally {
                $sw.Stop()
            }

            return [pscustomobject]@{
                SampledAtUtc = (Get-Date).ToUniversalTime().ToString("o")
                Outcome      = $outcome
                TtftMs       = $ttft
                LatencyMs    = $sw.ElapsedMilliseconds
                OutputChars  = $chars
                Error        = $errorText
            }
        }

        $jobs = @()
        for ($i = 0; $i -lt $Samples; $i++) {
            while (@($jobs | Where-Object { $_.State -eq "Running" }).Count -ge $Concurrency) {
                Start-Sleep -Milliseconds 150
            }
            $jobs += Start-Job -ScriptBlock $worker -ArgumentList $BaseUrl, $Model, $ReasoningEffort, $Prompt, $MaxTokens
        }
        $jobs | Wait-Job | Out-Null
        $results = @($jobs | Receive-Job)
        $failedJobs = @($jobs | Where-Object { $_.State -eq "Failed" })
        $jobs | Remove-Job -Force

        if ($failedJobs.Count -gt 0) {
            Fail "one or more sampling jobs failed before returning a result"
        }

        $samplesPath = Join-Path $OutDir ("local-latency-" + $Capability + ".csv")
        $results | Export-Csv -LiteralPath $samplesPath -NoTypeInformation -Encoding UTF8

        $ok = @($results | Where-Object { $_.Outcome -eq "success" -and $null -ne $_.TtftMs })
        if ($ok.Count -eq 0) { Fail "no successful latency samples were collected" }

        $ttfts = @($ok | ForEach-Object { [double]$_.TtftMs } | Sort-Object)
        $latencies = @($ok | ForEach-Object { [double]$_.LatencyMs } | Sort-Object)
        $p95Index = [Math]::Ceiling(0.95 * $ttfts.Count) - 1
        if ($p95Index -lt 0) { $p95Index = 0 }
        $latencyP95Index = [Math]::Ceiling(0.95 * $latencies.Count) - 1
        if ($latencyP95Index -lt 0) { $latencyP95Index = 0 }

        $summary.latencySampled = $true
        $summary.samples = $ok.Count
        $summary.ttftP50Ms = [int]$ttfts[([Math]::Floor(($ttfts.Count - 1) / 2))]
        $summary.ttftP95Ms = [int]$ttfts[$p95Index]
        $summary.latencyP50Ms = [int]$latencies[([Math]::Floor(($latencies.Count - 1) / 2))]
        $summary.latencyP95Ms = [int]$latencies[$latencyP95Index]
        $summary.successCount = $ok.Count
        $summary.errorCount = @($results | Where-Object { $_.Outcome -ne "success" }).Count
        $summary.samplesCsv = $samplesPath
        Write-Step ("latency: ttft p50=" + $summary.ttftP50Ms + " ms p95=" + $summary.ttftP95Ms + " ms; total p50=" + $summary.latencyP50Ms + " ms p95=" + $summary.latencyP95Ms + " ms")
        Write-Step "raw samples: $samplesPath"
    }
}

# ------------------------------------------------------------------------------
# Summary
# ------------------------------------------------------------------------------
$summaryPath = Join-Path $OutDir ("local-test-summary-" + $Capability + ".json")
($summary | ConvertTo-Json -Depth 5) | Set-Content -LiteralPath $summaryPath -Encoding UTF8
Write-Step "summary: $summaryPath"

if (-not $summary.functionalPassed -and -not $SkipFunctional) {
    Fail "functional checks did not pass"
}

Write-Step "DONE"
