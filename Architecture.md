# 🏗️ WarmHouse SaaS MVP — Architecture & System Design Document

## 1. Project Overview
**WarmHouse** — это SaaS-экосистема Умного дома нового поколения. 
Проект переходит от устаревшей монолитной синхронной архитектуры к распределенной, асинхронной и масштабируемой микросервисной архитектуре.
**Главная цель MVP:** Предоставить пользователям возможность самостоятельного подключения (self-service) устройств от разных партнеров, просмотра телеметрии в реальном времени и управления устройствами с минимальной задержкой.

## 2. Architectural Principles
1. **MQTT over WebSockets (WSS):** Для доставки данных в реальном времени (телеметрия) и сверхбыстрого управления устройствами мобильное приложение подключается напрямую к IoT MQTT Брокеру по протоколу WSS.
2. **Protocol-Agnostic Commands:** Система не завязана только на MQTT. Выделенный `Command Service` абстрагирует протоколы устройств (MQTT, HTTP, Matter) и транслирует универсальные команды.
3. **Event-Driven via Shared Subscriptions:** В MVP отсутствует тяжелая шина сообщений (Kafka/RabbitMQ). Внутренний роутинг событий реализуется через сам MQTT Брокер (Mosquitto v2.0+) с использованием механизма Shared Subscriptions.
4. **Smart Endpoints, Dumb Pipes:** Устройства (Hardware) максимально "глупые" — они знают только адрес MQTT-брокера и свои топики. Вся бизнес-логика и ACL находятся на стороне облака.

## 3. Technology Stack
* **Backend Languages:** Go (Golang) для всех микросервисов.
* **IoT Messaging Broker:** Eclipse Mosquitto (скомпилированный с плагином `mosquitto-go-auth` для поддержки HTTP Webhooks).
* **Databases:** PostgreSQL (для реляционных данных) и TimescaleDB (для хранения Time-series телеметрии).
* **Communication:** REST/JSON (Client -> API Gateway), gRPC или внутренний HTTP REST (межсервисное взаимодействие), MQTT (брокер -> микросервисы).

---

## 4. System Components (Microservices)

### 4.1. Edge & Connectivity Layer
* **API Gateway (Go):** * Единая точка входа для классических HTTP/REST запросов от мобильного приложения.
  * Обрабатывает логин, выдачу JWT-токенов, роутинг запросов к внутренним сервисам.
* **IoT MQTT Broker (Mosquitto + go-auth plugin):**
  * Ядро обмена сообщениями.
  * Принимает TCP/TLS подключения от железа.
  * Принимает WSS подключения от клиентских приложений.
  * *Архитектурная особенность:* Использует HTTP Webhooks (через плагин) для делегирования проверки логинов и прав подписки (ACL) в микросервис авторизации.

### 4.2. Core Business Services (PostgreSQL)
* **Account Service (Go):**
  * Управляет пользователями, их профилями и структурой "Дом -> Комнаты".
* **Device Service (Go):**
  * Реестр устройств (Device Registry). Хранит связи "Устройство -> Пользователь".
  * Хранит метаданные устройств (протокол, тип, возможности).
* **Auth & ACL Service (Go):**
  * Выступает в роли Webhook-сервера для плагина `mosquitto-go-auth`. 
  * Отвечает на запросы аутентификации (проверяет JWT для WSS клиентов и токены устройств для TCP).
  * Отвечает на запросы авторизации/ACL (проверяет, имеет ли клиент право на публикацию/подписку к конкретному топику, сверяясь с `Device Service`).

### 4.3. Data & Action Services
* **Command Service (Go):**
  * Принимает стандартизированные запросы на управление (REST).
  * Обогащает их данными из `Device Service` (узнает транспортный протокол устройства).
  * Отправляет команду: делает MQTT Publish в брокер ИЛИ прямой HTTP POST запрос на устройство.
* **Telemetry Service (Go) + TimescaleDB:**
  * Подписан на системный MQTT-топик со всей телеметрией (Shared Subscription, например `$share/telemetry_group/devices/+/telemetry`).
  * Парсит сырые данные, агрегирует и сохраняет в базу временных рядов (TimescaleDB) для построения графиков.
* **Scenario Service (Go):**
  * Движок автоматизации (Rules Engine). 
  * Слушает поток телеметрии из брокера, проверяет пользовательские правила ("Если температура < 20") и при срабатывании отправляет триггер в `Command Service`.

---

## 5. Core Data Flows

### Flow 1: Device Provisioning (Подключение нового устройства)
1. Пользователь сканирует QR-код устройства в приложении.
2. Приложение отправляет `POST /api/v1/devices` через `API Gateway`.
3. Запрос маршрутизируется в `Device Service`.
4. `Device Service` привязывает `DeviceID` к текущему `UserID` в PostgreSQL.
5. Устройство получает настройки сети (локально) и подключается к `Mosquitto`.

### Flow 2: Real-time Telemetry (Просмотр метрик)
1. Приложение запрашивает у `API Gateway` временный JWT и IP адрес брокера.
2. Приложение открывает WSS соединение с `Mosquitto` и подписывается на топик `clients/{user_id}/devices/+/telemetry`.
3. Плагин `mosquitto-go-auth` делает HTTP Webhook запрос в `Auth & ACL Service` для проверки прав.
4. Датчик публикует температуру по MQTT.
5. `Mosquitto` мгновенно (Push) доставляет эти данные по WSS в мобильное приложение.
6. Параллельно `Telemetry Service` вычитывает эти же данные и пишет их в БД для истории.

### Flow 3: Commanding (Управление устройством - HTTP Path)
*Используется для интеграций, голосовых ассистентов или HTTP-устройств.*
1. Клиент делает `POST /api/v1/commands` с телом `{"device_id": "123", "action": "turn_on"}`.
2. `Command Service` запрашивает у `Device Service` протокол устройства.
3. Если это MQTT-устройство, `Command Service` делает внутренний MQTT Publish в топик `devices/123/commands`.
4. Брокер доставляет команду на устройство.

---

## 6. Instructions for AI Agent
* **API Design:** При проектировании контрактов используй стандарт **OpenAPI 3.0** для REST интерфейсов (взаимодействие с API Gateway) и **AsyncAPI** для описания MQTT топиков и payload'ов.
* **Naming Conventions:** Используй `snake_case` для JSON payload'ов и URL параметров.
* **Statelessness:** Все HTTP микросервисы должны быть stateless (без сохранения состояния), состояние хранится только в PostgreSQL/TimescaleDB.
* **Security:** Все API эндпоинты (кроме `/login`, `/register` и внутренних Webhooks для Mosquitto) требуют наличия `Authorization: Bearer <JWT>` заголовка.
* **Mosquitto Webhooks:** `Auth & ACL Service` должен реализовывать контракты HTTP POST запросов, ожидаемые плагином `mosquitto-go-auth` (в частности `/auth` и `/acl` эндпоинты).