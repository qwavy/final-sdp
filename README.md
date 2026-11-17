# Parcel Hub

Приложение для пункта выдачи заказов и курьерской доставки.
Сотрудник принимает посылку, считает стоимость, назначает курьера,
отслеживает статусы и в конце дня выгружает отчёт.

**Сквозной сценарий:** создать заказ → выбрать способ доставки и доп. услуги →
сравнить цену по двум стратегиям → назначить курьера → провести по статусам
(клиенту уходит SMS) → посмотреть и выгрузить отчёт за день.

## Требования

- Go 1.25+
- Node.js 20+ (только для фронтенда)

## Запуск

Сначала соберите фронтенд (один раз и после каждого изменения фронтенда):

```bash
cd frontend && npm install && npm run build
```

Затем запустите сервер. Он отдаёт и API, и собранный фронтенд:

```bash
cd backend && go run ./cmd/server
```

Откройте http://localhost:8088.

Режим разработки фронтенда с горячей перезагрузкой: сервер Go запущен как выше, а в другом терминале:

```bash
cd frontend && npm run dev
```

Vite (http://localhost:5173) перенаправляет запросы `/api` на Go-сервер.

### Тестовые данные

При первом запуске (папка `backend/data` пустая) автоматически создаются
5 клиентов, 4 курьера, 4 точки выдачи и 7 заказов в разных статусах.
Чтобы начать заново, удалите `backend/data` и перезапустите сервер.

Флаги сервера: `-addr :8088`, `-data data`, `-static ../frontend/dist`.

### Тесты

```bash
cd backend && go test ./...
```

## Архитектура

Гексагональная архитектура (ports & adapters) + DDD-lite + CQS.

```
backend/
  cmd/server/              запуск, сборка зависимостей, тестовые данные
  internal/
    domain/                сущности, правила, паттерны предметной области
    application/
      facade.go            Facade — единая точка входа для HTTP
      port/                интерфейсы хранилища и уведомлений
      events/              Observer — Publisher и наблюдатели
      use_cases/
        command/           команды: меняют данные (CQS)
        query/             запросы: только читают данные (CQS)
    adapters/
      in/http/             REST API
      out/storage/         JSON-файлы
      out/notify/          Adapter — SMS-шлюз
frontend/                  React + TypeScript + shadcn/ui
```

Зависимости направлены внутрь: `adapters → application → domain`.
Domain не знает ни о HTTP, ни о файлах.

## Паттерны

| Паттерн | Тип | Какую проблему решает | Главные типы | Как показать |
|---|---|---|---|---|
| Builder | порождающий | У заказа много полей, часть зависит от способа доставки; заказ должен создаваться только целиком корректным | `domain.OrderBuilder` | Отправить пустую форму заказа — все ошибки сразу по полям |
| Factory Method | порождающий | Способы доставки отличаются тарифом, лимитом веса и цепочкой статусов | `DeliveryCreator`, `DeliveryMethod`, `DoorCreator` … | Заказ «до двери» не проходит через «Ждёт в точке выдачи»; постамат не принимает > 15 кг |
| Decorator | структурный | Доп. услуги в любой комбинации добавляются к цене | `PriceComponent`, `basePrice`, `insuranceExtra`, `fragileExtra`, `expressExtra` | Отмечать услуги в форме — строки расшифровки и итог меняются |
| Facade | структурный | HTTP не должен знать о командах, запросах и хранилище | `application.DeliveryFacade` | Любой обработчик в `adapters/in/http/handlers.go` вызывает один метод фасада |
| Adapter | структурный | Сторонний SMS-шлюз имеет несовместимый интерфейс | `port.Notifier` ← `notify.SmsNotifier` → `notify.SmsGateway` | Сменить статус — строка в `data/sms_outbox.txt` и запись в журнале |
| Strategy | поведенческий | Пользователь выбирает способ расчёта цены и способ выбора курьера | `PricingStrategy` (`WeightPricing`, `DistancePricing`), `CourierSelector` (`Nearest…`, `LeastLoaded…`) | В форме две цены рядом; в заказе выбор «Ближайший / Наименее загруженный» |
| Observer | поведенческий | Смена статуса должна вызывать несколько независимых реакций | `events.Publisher`, `Observer`, `ActivityRecorder`, `CustomerNotifier` | Сменить статус — в журнале появляются событие и SMS, строка в `data/sms_outbox.txt` |

## API (кратко)

| Метод | Путь | Что делает |
|---|---|---|
| GET | `/api/options` | справочники для форм |
| GET/POST | `/api/orders` | список (`?status=&q=`) / создание |
| POST | `/api/orders/quote` | расчёт цены по всем стратегиям |
| GET/PUT/DELETE | `/api/orders/{id}` | заказ |
| POST | `/api/orders/{id}/assign` | назначить курьера `{"selector":"nearest"}` |
| POST | `/api/orders/{id}/status` | сменить статус `{"status":"in_transit"}` |
| GET/POST/PUT/DELETE | `/api/customers`, `/api/couriers` | клиенты и курьеры |
| GET | `/api/pickup-points`, `/api/activity` | точки выдачи, журнал |
| GET | `/api/reports/daily?date=` | отчёт за день |
| GET | `/api/reports/daily/export?format=csv\|json` | выгрузка (копия в `data/exports`) |

Ошибки: `422` — ошибки ввода по полям, `404` — не найдено, `409` — нарушено правило
(например, недопустимая смена статуса), `400` — некорректный запрос.

## Документы для сдачи

- [docs/report/report.pdf](docs/report/report.pdf) — отчёт (PDF). Исходник — `docs/report/report.html`.
- [docs/uml/](docs/uml) — UML: диаграммы классов и последовательности (Mermaid `.mmd` + `.svg`).
- [docs/PROJECT.md](docs/PROJECT.md) — проект по ТЗ, [docs/CODE.md](docs/CODE.md) — разбор кода.
