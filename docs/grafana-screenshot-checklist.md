# Чеклист скриншотов Grafana для диплома

## Настройки перед съемкой
- Время дашборда: `Last 30 minutes`.
- Refresh: `5s`.
- Для каждого сценария снимай:
  - 1 скрин в середине `steady-state`.
  - 1 скрин в конце сценария (видна динамика и завершение).
- Имена файлов:
  - `S1_registration_*.png`
  - `S2_call_setup_*.png`
  - `S3_media_stress_*.png`

## Обязательные скриншоты (минимум 8)

### S1: Registration Storm
1. **SIP Requests/Statuses**
   - Панель с `rate(voip_worker_sip_requests_total)`
   - Видно объем REGISTER и коды ответа.
2. **Registration Delay**
   - Панель/квантиль для `voip_worker_registration_delay_seconds`
   - Зафиксировать p95.

### S2: Call Setup Rate
3. **ASR + NER**
   - `voip_worker_asr_ratio`, `voip_worker_ner_ratio`.
4. **Session Delay (SRD p95)**
   - `histogram_quantile(0.95, ... voip_worker_session_request_delay_seconds_bucket ...)`.
5. **Active Calls + SIP Throughput**
   - Корреляция `voip_worker_active_calls_current` и SIP request rate.

### S3: Media Stress
6. **RTP Jitter**
   - `voip_worker_rtp_jitter_ms`.
7. **RTP Packet Loss**
   - `voip_worker_rtp_packet_loss_pct`.
8. **RTP MOS**
   - `voip_worker_rtp_mos_estimated`.

## Рекомендуемые дополнительные (еще 2-3)
9. **SIP Retries**
   - `rate(voip_worker_sip_retries_total[1m])`.
10. **Точка деградации**
   - Скрин, где SLA начинает отклоняться (или фиксируется устойчивость при целевом CPS).

## Что подписать под каждым скриншотом
- Сценарий (`S1/S2/S3`).
- Параметры (`CPS`, длительность, users).
- Ключевой вывод из графика (1-2 предложения).

## Куда вставлять в текст
- Глава по тестированию:
  - сначала S1 (сигнализация регистрации),
  - потом S2 (сеанс/задержки),
  - затем S3 (медиа QoS).
- Итоговая таблица SLA:
  - ASR, NER, SRD p95, packet loss, jitter, MOS.
