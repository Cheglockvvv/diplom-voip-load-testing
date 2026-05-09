# План практического тестирования

## 1. Цель
Проверить работоспособность фреймворка и определить предельную нагрузку для SIP/RTP на тестовом стенде.

## 2. Стенд
- `asterisk` как DUT (Device Under Test)
- `controller` для управления прогонами
- `worker` для генерации SIP/RTP
- `prometheus` и `grafana` для мониторинга

## 3. Фазы каждого теста
- `ramp_up`: плавный рост CPS
- `steady_state`: удержание целевого CPS
- `ramp_down`: плавное снижение CPS

## 4. Набор тестов
1. `registration_storm` — устойчивость к всплеску REGISTER
2. `call_setup_rate` — производительность сигнализации INVITE
3. `media_stress` — влияние RTP на QoS

## 5. SLA критерии
- `ASR >= 0.99`
- `NER >= 0.99`
- `SRD P95 <= 0.2s`
- `RTP packet loss < 1%`
- `MOS >= 4.0`
