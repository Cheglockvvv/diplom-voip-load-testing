$ErrorActionPreference = "Stop"

function Get-MetricSnapshot {
    $raw = (Invoke-WebRequest -UseBasicParsing http://localhost:8081/metrics).Content
    $snapshot = @{
        sip       = @{}
        retries   = @{}
        rrd_sum   = 0.0
        rrd_count = 0.0
        srd_sum   = 0.0
        srd_count = 0.0
        asr       = 0.0
        ner       = 0.0
        loss      = 0.0
        jitter    = 0.0
        mos       = 0.0
    }
    foreach ($line in ($raw -split "`n")) {
        $l = $line.Trim()
        if ($l -match '^voip_worker_sip_requests_total\{method="([^"]+)",status="([^"]+)"\}\s+([0-9eE\+\-\.]+)$') {
            $snapshot.sip["$($matches[1])|$($matches[2])"] = [double]$matches[3]
        } elseif ($l -match '^voip_worker_sip_retries_total\{method="([^"]+)"\}\s+([0-9eE\+\-\.]+)$') {
            $snapshot.retries[$matches[1]] = [double]$matches[2]
        } elseif ($l -match '^voip_worker_registration_delay_seconds_sum\s+([0-9eE\+\-\.]+)$') {
            $snapshot.rrd_sum = [double]$matches[1]
        } elseif ($l -match '^voip_worker_registration_delay_seconds_count\s+([0-9eE\+\-\.]+)$') {
            $snapshot.rrd_count = [double]$matches[1]
        } elseif ($l -match '^voip_worker_session_request_delay_seconds_sum\s+([0-9eE\+\-\.]+)$') {
            $snapshot.srd_sum = [double]$matches[1]
        } elseif ($l -match '^voip_worker_session_request_delay_seconds_count\s+([0-9eE\+\-\.]+)$') {
            $snapshot.srd_count = [double]$matches[1]
        } elseif ($l -match '^voip_worker_asr_ratio\s+([0-9eE\+\-\.]+)$') {
            $snapshot.asr = [double]$matches[1]
        } elseif ($l -match '^voip_worker_ner_ratio\s+([0-9eE\+\-\.]+)$') {
            $snapshot.ner = [double]$matches[1]
        } elseif ($l -match '^voip_worker_rtp_packet_loss_pct\s+([0-9eE\+\-\.]+)$') {
            $snapshot.loss = [double]$matches[1]
        } elseif ($l -match '^voip_worker_rtp_jitter_ms\s+([0-9eE\+\-\.]+)$') {
            $snapshot.jitter = [double]$matches[1]
        } elseif ($l -match '^voip_worker_rtp_mos_estimated\s+([0-9eE\+\-\.]+)$') {
            $snapshot.mos = [double]$matches[1]
        }
    }
    return $snapshot
}

function Diff-Map($after, $before) {
    $result = @{}
    $keys = @($before.Keys + $after.Keys | Sort-Object -Unique)
    foreach ($k in $keys) {
        $b = 0.0
        $a = 0.0
        if ($before.ContainsKey($k)) { $b = [double]$before[$k] }
        if ($after.ContainsKey($k)) { $a = [double]$after[$k] }
        $result[$k] = [math]::Round(($a - $b), 3)
    }
    return $result
}

function Wait-UntilIdle([int]$timeoutSec) {
    $deadline = (Get-Date).AddSeconds($timeoutSec)
    while ((Get-Date) -lt $deadline) {
        $st = Invoke-RestMethod -Uri "http://localhost:8080/status" -Method Get
        if ($st.state -eq "idle") {
            return $true
        }
        Start-Sleep -Seconds 3
    }
    return $false
}

$runs = @(
    @{ name = "LADDER_S2_CPS2"; scenario = "scenarios/onepc/s2_call_setup_cps2.yaml"; timeout = 240 },
    @{ name = "LADDER_S2_CPS4"; scenario = "scenarios/onepc/s2_call_setup_cps4.yaml"; timeout = 240 },
    @{ name = "LADDER_S2_CPS6"; scenario = "scenarios/onepc/s2_call_setup_cps6.yaml"; timeout = 240 },
    @{ name = "LADDER_S2_CPS8"; scenario = "scenarios/onepc/s2_call_setup_cps8.yaml"; timeout = 240 },
    @{ name = "LADDER_S3_CPS2"; scenario = "scenarios/onepc/s3_media_cps2.yaml"; timeout = 240 },
    @{ name = "SOAK_S3_15M"; scenario = "scenarios/onepc/s3_media_soak_15m.yaml"; timeout = 1200 }
)

$report = @{
    generated_at = (Get-Date).ToString("s")
    runs = @()
}

foreach ($run in $runs) {
    Write-Host "=== RUN $($run.name) ==="
    $before = Get-MetricSnapshot
    $start = Get-Date
    $runOutput = & go run ./cmd/cli run -controller http://localhost:8080 -scenario $run.scenario
    $ok = Wait-UntilIdle -timeoutSec $run.timeout
    if (-not $ok) {
        & go run ./cmd/cli stop -controller http://localhost:8080 | Out-Null
    }
    $end = Get-Date
    $after = Get-MetricSnapshot

    $rrdAvg = if (($after.rrd_count - $before.rrd_count) -gt 0) { (($after.rrd_sum - $before.rrd_sum) / ($after.rrd_count - $before.rrd_count)) } else { 0.0 }
    $srdAvg = if (($after.srd_count - $before.srd_count) -gt 0) { (($after.srd_sum - $before.srd_sum) / ($after.srd_count - $before.srd_count)) } else { 0.0 }

    $report.runs += [ordered]@{
        name                          = $run.name
        scenario_file                 = $run.scenario
        started_at                    = $start.ToString("s")
        ended_at                      = $end.ToString("s")
        elapsed_seconds               = [int]($end - $start).TotalSeconds
        completed                     = $ok
        run_command_output            = ($runOutput -join "`n")
        sip_requests_delta            = (Diff-Map $after.sip $before.sip)
        sip_retries_delta             = (Diff-Map $after.retries $before.retries)
        registration_delay_avg_seconds = [math]::Round($rrdAvg, 6)
        session_delay_avg_seconds     = [math]::Round($srdAvg, 6)
        asr_ratio_after               = [math]::Round($after.asr, 6)
        ner_ratio_after               = [math]::Round($after.ner, 6)
        rtp_packet_loss_after         = [math]::Round($after.loss, 6)
        rtp_jitter_ms_after           = [math]::Round($after.jitter, 6)
        rtp_mos_after                 = [math]::Round($after.mos, 6)
    }
}

$jsonPath = "docs/onepc-benchmark-results.json"
$report | ConvertTo-Json -Depth 8 | Set-Content -Path $jsonPath -Encoding UTF8

$md = @()
$md += "# Single-PC benchmark results"
$md += ""
$md += "Generated at: $($report.generated_at)"
$md += ""
$md += "| Run | Scenario | Elapsed (s) | Completed | ASR | NER | SRD avg (s) | RTP Loss | RTP Jitter | MOS |"
$md += "|---|---|---:|:---:|---:|---:|---:|---:|---:|---:|"
foreach ($r in $report.runs) {
    $md += "| $($r.name) | $($r.scenario_file) | $($r.elapsed_seconds) | $($r.completed) | $($r.asr_ratio_after) | $($r.ner_ratio_after) | $($r.session_delay_avg_seconds) | $($r.rtp_packet_loss_after) | $($r.rtp_jitter_ms_after) | $($r.rtp_mos_after) |"
}
$md += ""
$md += "Raw per-run SIP status deltas are available in \`$jsonPath\`."
$mdPath = "docs/onepc-benchmark-results.md"
$md -join "`n" | Set-Content -Path $mdPath -Encoding UTF8

Write-Host "Saved $jsonPath and $mdPath"
