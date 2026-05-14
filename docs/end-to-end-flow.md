# Процесс от нуля до конца нагрузочного тестирования

## Этап 0: Подготовка
1. Клонировать репозиторий.
2. Проверить версии:
   - Go 1.23+
   - Docker Desktop (если запускается контейнерный стенд)
3. Выполнить локальную валидацию:
   - `go test ./...`
   - `go build ./...`

## Этап 1: Конфигурация сценария
1. Выбрать профиль:
   - `scenarios/registration_storm.yaml`
   - `scenarios/call_setup_rate.yaml`
   - `scenarios/media_stress.yaml`
2. Настроить параметры:
   - `cps`, `duration_seconds`, `users`
   - `sip_timeout_ms`, `sip_retry_attempts`
   - `target.host`, `target.sip_port`, `target.rtp_port`

## Этап 2: Запуск инфраструктуры
1. Поднять сервисы:
   - `docker compose -f deploy/docker-compose.yml up --build -d`
2. Проверить health endpoints:
   - controller: `GET /health`
   - worker: `GET /health`
3. Убедиться, что доступны:
   - Prometheus (`:9090`)
   - Grafana (`:3000`)

## Этап 3: Запуск теста
1. Отправить команду запуска:
   - `go run ./cmd/cli run -controller http://localhost:8080 -scenario <path>`
2. Получить `run_id` и зафиксировать в отчете.
3. Мониторить:
   - `go run ./cmd/cli status ...`
   - `go run ./cmd/cli watch-status ...`
   - панели Grafana по SIP/RTP/QoS.

## Этап 4: Сбор результатов
1. По окончании или вручную остановить:
   - `go run ./cmd/cli stop -controller http://localhost:8080`
2. Снять метрики:
   - ASR, NER, SRD P95
   - RTP packet loss, jitter, MOS
3. Заполнить:
   - `docs/results-template.md`
4. Сохранить скриншоты Grafana.

## Этап 5: Анализ и выводы
1. Сравнить метрики с SLA-порогами.
2. Определить точку деградации.
3. Сформулировать:
   - лимиты стенда;
   - рекомендации по масштабированию;
   - план следующего шага (multi-host/облако).
