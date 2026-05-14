# Финальный чеклист перед защитой

## 1) Подготовка стенда
- `docker compose -f deploy/docker-compose.yml up --build -d`
- Проверить:
  - `http://localhost:8080/health`
  - `http://localhost:8081/health`
  - `http://localhost:9090`
  - `http://localhost:3000`

## 2) Прогоны сценариев
1. `registration_storm`
   - `go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/registration_storm.yaml`
2. `call_setup_rate`
   - `go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/call_setup_rate.yaml`
3. `media_stress`
   - `go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/media_stress.yaml`

Для онлайн-мониторинга статуса:
- `go run ./cmd/cli watch-status -controller http://localhost:8080`

Для остановки:
- `go run ./cmd/cli stop -controller http://localhost:8080`

## 3) Что фиксировать в отчете
- Скриншоты Grafana:
  - SIP request rate и SIP retries
  - ASR/NER
  - SRD p95
  - RTP jitter/loss/MOS
- Заполнить `docs/results-template.md`:
  - параметры прогона
  - P50/P95
  - точка деградации SLA
  - итоговый вывод

## 4) SLA контроль
- `ASR >= 0.99`
- `NER >= 0.99`
- `SRD P95 <= 0.2s`
- `RTP packet loss < 1%`
- `MOS >= 4.0`

## 5) Артефакты к защите
- Репозиторий с кодом и историей коммитов.
- Скриншоты дашбордов.
- Заполненный шаблон результатов.
- Краткое объяснение ограничений Docker-стенда и плана масштабирования на multi-host.
