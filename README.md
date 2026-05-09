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
3. Запустить сценарий:
   - `go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/registration_storm.yaml`
4. Остановить:
   - `go run ./cmd/cli stop -controller http://localhost:8080`

## Метрики
- Worker metrics: `http://localhost:8081/metrics`
- Controller metrics: `http://localhost:8080/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (admin/admin)

Основные метрики:
- `voip_worker_sip_requests_total`
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

## Примечания
- Файл `api/control.proto` добавлен как контракт для следующего этапа миграции controller<->worker на полноценный gRPC streaming.
- В текущем MVP управление реализовано через HTTP API для упрощения запуска без генерации protobuf кода.

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

## Commit strategy
- Коммиты делать по смысловым блокам, 1 блок = 1 commit:
  - reliability/runtime fixes
  - tests + CI
  - docs/process
- Не смешивать рефакторинг, функциональные изменения и инфраструктурные правки в одном commit.
- Держать сообщения коммитов в формате: короткий заголовок + 1-2 строки зачем изменение.
