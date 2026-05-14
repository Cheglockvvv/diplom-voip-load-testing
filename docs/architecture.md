# Архитектура приложения

## Компоненты и связи

```mermaid
flowchart LR
  cli[CLI]
  controller[Controller HTTP API]
  grpcControl[gRPC ControlService]
  worker[Worker Engine]
  sipTarget[SIP Server / IP-PBX]
  rtpTarget[RTP Endpoint]
  metrics[Prometheus]
  grafana[Grafana]

  cli --> controller
  controller --> grpcControl
  grpcControl --> worker
  worker --> sipTarget
  worker --> rtpTarget
  worker --> metrics
  controller --> metrics
  metrics --> grafana
```

## Логические модули
- `cmd/cli` — запуск/остановка сценариев, запрос статуса и поток статуса.
- `cmd/controller` — HTTP API для пользователя, gRPC-клиент к worker.
- `cmd/worker` — gRPC сервер управления, HTTP `/metrics` и `/debug/pprof`.
- `internal/worker` — ядро генерации SIP/RTP, FSM вызова, ретраи/таймауты.
- `internal/sip` — сборка/парсинг SIP, Digest-auth.
- `internal/metrics` — публикация counters/gauges/histograms.
- `internal/qos` и `internal/rtp` — оценка MOS и генерация RTP.

## Поток выполнения сценария
```mermaid
sequenceDiagram
  participant U as User/CLI
  participant C as Controller
  participant W as Worker
  participant S as SIP Server
  participant P as Prometheus/Grafana

  U->>C: POST /run (scenario)
  C->>W: gRPC StartScenario
  W-->>C: run_id
  C-->>U: 202 Accepted + run_id

  loop During scenario
    W->>S: SIP REGISTER/INVITE/BYE + RTP
    W->>P: /metrics updates
    U->>C: GET /status or /status/stream
    C->>W: gRPC GetStatus/StreamStatus
    C-->>U: state updates
  end

  U->>C: POST /stop
  C->>W: gRPC StopScenario
  W-->>C: stopped
  C-->>U: 200 OK
```
