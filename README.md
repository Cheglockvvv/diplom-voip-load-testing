# VoIP Load Testing Framework (Diploma MVP)

MVP фреймворк для нагрузочного тестирования SIP/RTP с метриками QoS.

## Состав
- `controller` — управление запусками/остановкой сценариев.
- `worker` — генерация нагрузки и расчет метрик.
- `asterisk` — тестируемая IP-АТС.
- `prometheus` + `grafana` — мониторинг.

## Быстрый старт
1. Поднять стенд:
   - `docker compose -f deploy/docker-compose.yml up --build -d`
2. Проверить health:
   - `http://localhost:8080/health` (controller)
   - `http://localhost:8081/health` (worker)
   - gRPC control plane: `localhost:19091` (worker)
3. Запустить сценарий:
   - `go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/registration_storm.yaml`
4. Остановить:
   - `go run ./cmd/cli stop -controller http://localhost:8080`
5. Проверить состояние выполнения:
   - `go run ./cmd/cli status -controller http://localhost:8080`

## Метрики
- Worker metrics: `http://localhost:8081/metrics`
- Controller metrics: `http://localhost:8080/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (admin/admin)

Основные метрики:
- `voip_worker_sip_requests_total`
- `voip_worker_sip_retries_total`
- `voip_worker_registration_delay_seconds`
- `voip_worker_session_request_delay_seconds`
- `voip_worker_asr_ratio`, `voip_worker_ner_ratio`
- `voip_worker_rtp_jitter_ms`, `voip_worker_rtp_packet_loss_pct`, `voip_worker_rtp_mos_estimated`

## Сценарии
- `scenarios/registration_storm.yaml`
- `scenarios/call_setup_rate.yaml`
- `scenarios/media_stress.yaml`

## Документация
- План тестов: `docs/test-plan.md`
- Шаблон результатов: `docs/results-template.md`
- Финальный чеклист: `docs/final-runbook.md`
- Архитектура: `docs/architecture.md`
- End-to-end процесс: `docs/end-to-end-flow.md`
- Актуальные результаты тестирования: `docs/testing-results-2026-05-14.md`
- Чеклист скриншотов Grafana: `docs/grafana-screenshot-checklist.md`

## Примечания
- Управление `controller -> worker` выполняется через gRPC (`ControlService`) с методами `StartScenario`, `StopScenario`, `GetStatus`, `StreamStatus`.
- Для упрощения локальной разработки применяется JSON codec поверх gRPC (без генерации protobuf-кода на текущем этапе).

## Production workflow
- Локальная проверка перед каждым push:
  - `go test ./...`
  - `go build ./...`
- В CI выполняются:
  - форматирование (`gofmt -l`)
  - тесты
  - сборка
- Для повторяемых команд использовать `Makefile`:
  - `make test`
  - `make build`
  - `make run-cli-s1|run-cli-s2|run-cli-s3`

## Runtime operations
- `POST /run` теперь возвращает `run_id`, что позволяет трассировать конкретный прогон.
- `GET /status` на controller проксирует состояние worker (`idle|running`) и текущий `run_id`.
- `GET /status/stream` на controller отдает поток NDJSON-событий статуса из gRPC `StreamStatus`.
- Controller и Worker поддерживают graceful shutdown по `SIGINT/SIGTERM` с таймаутом 10 секунд.
- В сценариях можно настраивать SIP транспорт:
  - `sip_timeout_ms` (таймаут SIP транзакции)
  - `sip_retry_attempts` (количество попыток отправки SIP запроса)

## Commit strategy
- Коммиты делать по смысловым блокам, 1 блок = 1 commit:
  - reliability/runtime fixes
  - tests + CI
  - docs/process
- Не смешивать рефакторинг, функциональные изменения и инфраструктурные правки в одном commit.
- Держать сообщения коммитов в формате: короткий заголовок + 1-2 строки зачем изменение.
