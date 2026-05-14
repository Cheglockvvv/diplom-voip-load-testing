# Diploma Sync Pack

Этот файл — единая точка синхронизации для генерации текста диплома.

## 1) Что реализовано
- Нагрузочный framework на Go: `controller` + `worker` + `cli`.
- gRPC control plane между `controller` и `worker`.
- SIP/RTP генерация и QoS метрики.
- Контейнерный стенд: Asterisk + Prometheus + Grafana.
- Автопрогоны сценариев и сбор метрик в JSON/Markdown.

## 2) Архитектура (кратко)
- CLI отправляет команды в Controller (`/run`, `/stop`, `/status`, `/status/stream`).
- Controller вызывает Worker по gRPC (`StartScenario`, `StopScenario`, `GetStatus`, `StreamStatus`).
- Worker генерирует SIP/RTP трафик в целевой Asterisk и экспортирует метрики.
- Prometheus собирает метрики, Grafana отображает дашборд.

Подробно и с диаграммами:
- `docs/architecture.md`
- `docs/end-to-end-flow.md`

## 3) Ключевые результаты тестирования

### Базовые сценарии (S1/S2/S3)
- Файл: `docs/testing-metrics-live.json`
- Ключевые наблюдения:
  - S1 стабильно отрабатывает (REGISTER challenge flow).
  - S2/S3 на single-PC упираются в локальные ограничения (высокий INVITE timeout при росте нагрузки).

### Усиление single-PC (ladder + soak)
- Файлы:
  - `docs/onepc-benchmark-results.json`
  - `docs/onepc-benchmark-results.md`
- Прогоны:
  - `LADDER_S2_CPS2/4/6/8`
  - `LADDER_S3_CPS2`
  - `SOAK_S3_15M`
- Все прогоны завершились корректно, есть воспроизводимые артефакты.

## 4) Ограничения и корректная интерпретация
- Стенд на одном ПК валиден для:
  - проверки функциональной корректности,
  - сравнительного анализа сценариев,
  - демонстрации методики и observability.
- Абсолютные значения throughput/SLA на single-host не эквивалентны production multi-host.
- В дипломе корректно формулировать это как ограничение экспериментального стенда.

## 5) Ключевые куски кода (для ссылки в тексте)
- gRPC контракт и client/server wiring:
  - `api/control/service.go`
  - `cmd/controller/main.go`
  - `cmd/worker/main.go`
- Control HTTP API:
  - `internal/controller/server.go`
- Ядро генерации и сценариев:
  - `internal/worker/runner.go`
- SIP сообщения и auth:
  - `internal/sip/messages.go`
  - `internal/sip/auth.go`
- Метрики:
  - `internal/metrics/metrics.go`

## 6) Примеры YAML сценариев

Базовые:
- `scenarios/registration_storm.yaml`
- `scenarios/call_setup_rate.yaml`
- `scenarios/media_stress.yaml`

Single-PC усиление:
- `scenarios/onepc/s2_call_setup_cps2.yaml`
- `scenarios/onepc/s2_call_setup_cps4.yaml`
- `scenarios/onepc/s2_call_setup_cps6.yaml`
- `scenarios/onepc/s2_call_setup_cps8.yaml`
- `scenarios/onepc/s3_media_cps2.yaml`
- `scenarios/onepc/s3_media_soak_15m.yaml`

## 7) Какие скриншоты Grafana вставлять

Файл чеклиста:
- `docs/grafana-screenshot-checklist.md`

Минимум 8:
1. SIP Requests/Statuses (S1)
2. Registration Delay p95 (S1)
3. ASR+NER (S2)
4. SRD p95 (S2)
5. Active Calls + Throughput (S2)
6. RTP Jitter (S3)
7. RTP Packet Loss (S3)
8. RTP MOS (S3)

Дополнительно:
9. SIP retries
10. Точка деградации/стабильности

## 8) Как запускать с нуля
- Финальный runbook: `docs/final-runbook.md`
- Команда one-PC benchmark:
  - `powershell -ExecutionPolicy Bypass -File scripts/run_onepc_benchmark.ps1`
  - или `make run-onepc-benchmark`

## 9) Что дать Gemini для текста диплома
- Этот файл (`docs/diploma-sync-pack.md`) как основной контекст.
- `docs/architecture.md` для раздела архитектуры.
- `docs/testing-metrics-live.json` и `docs/onepc-benchmark-results.json` для раздела результатов.
- `docs/grafana-screenshot-checklist.md` для списка иллюстраций.
