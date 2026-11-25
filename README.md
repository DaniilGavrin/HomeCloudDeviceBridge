# HomeCloud Device Bridge

**Слой интеграции физических устройств в экосистеме [ByteWizard HomeCloud](https://github.com/DaniilGavrin).**

HomeCloud Device Bridge — отдельный Go-процесс, который обнаруживает физические устройства, выполняет протокольное сопряжение, поддерживает подключения через драйверы и преобразует команды и события устройств в нормализованную модель HomeCloud.

Bridge запускается на том же узле, что и **HomeCloud Backend**. Пользовательский интерфейс взаимодействует только с Backend; прямого соединения UI ↔ Bridge нет.

> **Статус:** bootstrap-заглушка. Сейчас реализованы запуск процесса, локальный HTTP health endpoint и graceful shutdown. Backend WebSocket, реестр, discovery, pairing и драйверы ещё не реализованы.

## Архитектурные документы

- [Архитектура Device Bridge v1](docs/ARCHITECTURE_V1.md)

Нормативные спецификации: [Device Contracts v1](docs/DEVICE_CONTRACTS_V1.md), [Backend ↔ Bridge Protocol v1](docs/BRIDGE_PROTOCOL_V1.md), [Driver API v1](docs/DRIVER_API_V1.md), [Architecture v1](docs/ARCHITECTURE_V1.md).

## Основные обязанности

- Обнаружение устройств через первые интеграции Bluetooth LE и Wi-Fi/LAN; Zigbee, Matter и Thread не входят в первый этап.
- Обнаружение временных pairing candidates и передача их Backend.
- Выполнение протокольного сопряжения после того, как пользователь выбрал устройство в UI.
- Хранение локальной протокольной идентичности и mapping на HomeCloud `device_id` в собственном Bridge Device Registry.
- Выполнение команд через драйверы и нормализация событий/состояния.
- Восстановление соединений и повторная синхронизация после сбоев.

## Сопряжение устройств

HomeCloud использует полуавтоматическую модель:

1. Пользователь физически переводит устройство в режим сопряжения.
2. Bridge обнаруживает его и передаёт candidate через Backend в UI.
3. Пользователь выбирает нужное устройство.
4. Bridge автоматически выполняет протокольное сопряжение и регистрацию.
5. При повторном обнаружении уже сопряжённое устройство восстанавливается без повторного выбора, если его identity надёжно распознана.

Обнаружение само по себе не регистрирует устройство и не разрешает управление им.

## Разделение ответственности и реестров

| Компонент | Ответственность |
|---|---|
| **HomeCloud Backend** | Постоянный HomeCloud `device_id`, Physical Device Registry, Device Type/Capability validation, permissions, публичный API, маршрутизация команд |
| **HomeCloud Device Bridge** | Протокольная identity, Bridge Device Registry, discovery, pairing, runtime connections, драйверы, выполнение физических операций |
| **HomeCloud UI** | Выбор обнаруженного устройства, отображение модели Backend, отправка пользовательских команд |

Существуют два отдельных реестра физических устройств — в Backend и Bridge — и отдельный существующий `hc_devices`, который остаётся реестром WebSocket-клиентов Backend. Системный Bridge peer может присутствовать в `hc_devices`; физические устройства туда не записываются.

## Взаимодействие с Backend

Bridge подключается к локальному Backend по **WebSocket JSON v1 через loopback**. Отдельный публичный порт и отдельный Unix Socket / Windows Named Pipe для v1 не вводятся.

Backend управляет процессом Bridge как supervised child process. HTTP health endpoint используется для контроля процесса, WS registration/readiness — для подтверждения готовности управляющего канала, а закрытие stdin инициирует graceful shutdown.

## Модель драйверов

В v1 драйверы реализуются как встроенные Go-пакеты в одном бинарном файле Bridge. Динамическая загрузка исполняемых модулей не входит в первый этап.

## Принципы

- Локальная работа без обязательных облаков производителей.
- Самостоятельное размещение и отсутствие vendor lock-in.
- Явные границы Backend, Bridge и UI.
- Минимальные привилегии и безопасное хранение pairing credentials.
- Контрактная совместимость, reconnect, timeout и graceful shutdown.
- Низкие требования к CPU, RAM и сетевым ресурсам.

## Планируемые интеграции

| Интеграция | Область применения |
|---|---|
| Bluetooth Low Energy | Устройства с поддерживаемыми GATT-профилями (не универсальная поддержка любого BLE-устройства) |
| Wi-Fi / LAN | Устройства с поддерживаемыми локальными API/протоколами |
| Zigbee / Matter / Thread | Возможные последующие интеграции; не входят в первый этап |

Наличие интеграции в таблице обозначает направление разработки, а не уже реализованную поддержку.

## Разработка

Требуется Go **1.27.3** или совместимая версия.

### Сборка и проверка

```bash
go test -count=1 ./...
go vet ./...
go build -trimpath -o ./bin/HomeCloudDeviceBridge ./cmd/device-bridge
```

### Запуск заглушки

```bash
./bin/HomeCloudDeviceBridge
```

По умолчанию health endpoint доступен только локально: `http://127.0.0.1:8094/healthz`.
Адрес можно переопределить переменной окружения `HC_DEVICE_BRIDGE_HEALTH_ADDR`.
Процесс корректно завершает работу по SIGINT/SIGTERM или при закрытии stdin (EOF).

Ответ health endpoint обозначает только работоспособность процесса. Он **не** означает регистрацию в Backend или готовность управлять устройствами.

### CI/CD

- Pull Request в `main`: проверка модулей, тесты и `go vet`.
- Push в `main`: те же проверки, сборка Linux amd64, SHA-256 и commit-specific GitHub Pre-release.
- Release binary: `HomeCloudDeviceBridge-linux-amd64`.

GitHub Actions workflows находятся в `.github/workflows/`.

---

**Репозиторий:** [DaniilGavrin/HomeCloudDeviceBridge](https://github.com/DaniilGavrin/HomeCloudDeviceBridge)
