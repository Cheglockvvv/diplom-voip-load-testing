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

## Ограничения этого прогона
- Полный Docker-стенд не выполнен в этой сессии, так как `docker` отсутствовал в PATH.
- Для диплома рекомендуется повторить контейнерный прогон по `docs/final-runbook.md` и приложить скриншоты Grafana.

## Вывод
- Фреймворк функционально готов к демонстрации и защите:
  - управление сценариями стабильно,
  - SIP ядро покрыто интеграционными тестами,
  - наблюдаемость и поток статусов подтверждены,
  - артефакты для диплома подготовлены.
