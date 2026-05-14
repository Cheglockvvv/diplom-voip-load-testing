# Результаты тестирования (2026-05-14)

## Контекст
- Окружение: Windows 10 (локальный запуск).
- Репозиторий: `main`.
- Цель: подтвердить корректность ядра, control plane и финальных улучшений перед включением результатов в диплом.

## 1) Unit + Integration test suite

Команда:
- `go test ./...`

Результат:
- PASS для всех пакетов.
- Ключевые пакеты:
  - `internal/sip`
  - `internal/worker`
  - `internal/controller`

## 2) Протокольные тесты SIP/RTP ядра (verbose)

Команда:
- `go test -v ./internal/sip ./internal/worker`

Проверенные кейсы:
- Digest challenge parsing.
- Digest auth header с `qop=auth`.
- Registration flow: `REGISTER -> 401 -> REGISTER(auth) -> 200`.
- Call flow: `INVITE -> 180/183 -> 200 -> ACK -> BYE`.
- Ramp-up / steady / ramp-down вычисление CPS.

Результат:
- PASS для всех перечисленных кейсов.

## 3) Control plane smoke test (controller + worker + gRPC)

Локальный запуск на альтернативных портах:
- worker: HTTP `:48081`, gRPC `:49091`
- controller: HTTP `:48080` (подключен к worker gRPC)

Команды и результаты:
1. `cli status`:
   - `status: 200 OK`, `state: idle`
2. `cli run` (`registration_storm.yaml`):
   - `status: 202 Accepted`, `run_id: run-1`
3. `cli status` во время прогона:
   - `status: 200 OK`, `run_id: run-1`, `state: running`
4. `cli watch-status`:
   - получены stream-события (`running`, `status snapshot`)
5. `cli stop`:
   - `status: 200 OK`, `run_id: run-1`, `message: stopped`

Результат:
- gRPC control plane и status streaming работают корректно.

## 4) Полный Docker-прогон сценариев

Стенд:
- `docker compose -f deploy/docker-compose.yml up -d`
- Проверка health:
  - controller: `ok`
  - worker: `ok`
  - Prometheus: `Healthy`
  - Grafana API: `database=ok`

Фактические прогоны:
1. `registration_storm.yaml`:
   - старт: `22:56:24`
   - завершение: `22:58:24`
   - длительность: `120s`
2. `call_setup_rate.yaml`:
   - старт: `22:58:24`
   - завершение: `23:01:25`
   - длительность: `180s`
3. `media_stress.yaml`:
   - старт: `23:01:25`
   - завершение: `23:04:26`
   - длительность: `180s`

Сводные метрики из Prometheus после прогонов:
- `voip_worker_asr_ratio = 0`
- `voip_worker_ner_ratio = 0`
- `SRD p95 ~= 0.0095s`
- `voip_worker_rtp_packet_loss_pct = 0`
- `voip_worker_rtp_jitter_ms = 0`
- `voip_worker_rtp_mos_estimated = 0`
- SIP-коды за окно 30m:
  - `REGISTER 401 ~= 23862`
  - `INVITE 401 ~= 11798`

Интерпретация:
- Сценарии отрабатывают в полном цикле и завершаются автоматически.
- Наблюдаемость и сбор метрик рабочие.
- Для получения целевых SLA-значений ASR/NER в сценариях вызовов нужно донастроить сторону Asterisk (в текущем стенде доминируют challenge-ответы `401`).

## Вывод
- Фреймворк функционально готов к демонстрации и защите:
  - управление сценариями стабильно,
  - SIP ядро покрыто интеграционными тестами,
  - наблюдаемость и поток статусов подтверждены,
  - артефакты для диплома подготовлены.
