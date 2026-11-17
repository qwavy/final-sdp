# Parcel Hub — разбор кода

Что сделано по ТЗ и зачем паттерны — в [PROJECT.md](PROJECT.md).
Здесь — **как устроен код**, простыми словами.

---

## 1. Папки

```
backend/
  cmd/server/main.go     ← запуск: соединяет все части
  cmd/server/seed.go     ← тестовые данные при первом запуске
  internal/
    domain/              ← правила и паттерны (Builder, Factory Method, Strategy, Decorator)
    application/
      facade.go          ← Facade: единственный вход для HTTP
      port/              ← интерфейсы: «что нам нужно снаружи»
      events/            ← Observer
      use_cases/
        command/         ← команды: МЕНЯЮТ данные
        query/           ← запросы: ТОЛЬКО ЧИТАЮТ
    adapters/
      in/http/           ← REST API
      out/storage/       ← JSON-файлы
      out/notify/        ← SMS (Adapter)
```

**Гексагональная архитектура:** в центре — `domain` и `application`, по краям — адаптеры.
Внутренний слой никогда не импортирует внешний: `adapters → application → domain`.

**CQS:** команды меняют данные и возвращают только ошибку (или ID нового объекта).
Запросы только читают и возвращают данные для экрана.

---

## 2. Как проходит один запрос

Пользователь нажал «→ В пути»:

```
браузер:  POST /api/orders/ORD-1/status  {"status": "in_transit"}
   │
   ▼
http/handlers.go   changeStatus      → читает JSON
   ▼
facade.go          ChangeOrderStatus
   ├─ command  ChangeStatus
   │     1. orders.Get(id)            — достать из файла
   │     2. order.ChangeStatus(...)   — domain проверяет правило
   │     3. orders.Save(order)        — записать в файл
   │     4. publisher.Notify(event)   — Observer: журнал + SMS
   └─ query    Get(id)                — прочитать заказ для ответа
   ▼
браузер получает JSON заказа, показывает уведомление и обновляет таблицы
```

Если правило нарушено («Создан» → «Выдан»), на шаге 2 вернётся ошибка,
ничего не сохранится, а браузер получит `409` с текстом ошибки.

---

## 3. Domain — правила

| Файл | Что внутри |
|---|---|
| `order.go` | заказ, статусы, `AssignCourier`, `ChangeStatus` |
| `order_builder.go` | **Builder** |
| `delivery.go` | **Factory Method** — способы доставки |
| `pricing.go` | **Strategy** — расчёт цены |
| `courier_selection.go` | **Strategy** — выбор курьера |
| `price.go` | **Decorator** — доп. услуги |
| `customer.go`, `courier.go`, `pickup_point.go` | клиент, курьер, точка выдачи |
| `errors.go` | ошибки: `ValidationError`, `NotFound`, `RuleViolation` |
| `geo.go` | районы и расстояния |
| `common.go` | `Money`, генерация ID, телефон |

### Builder

```go
order, err := domain.NewOrderBuilder().
    ForCustomer("CUS-1").
    Parcel("Книги", 2.5, 10000).
    Delivery(domain.DeliveryLocker).
    ToPickupPoint(point).
    PricedBy("by_weight").
    WithExtras("fragile").
    Build(time.Now())
```

Каждый метод запоминает значение и возвращает builder — поэтому можно писать цепочкой.
Все проверки — в `Build()`. Он собирает **все** ошибки сразу
(`ValidationError` — это `map[поле]текст`), поэтому форма подсвечивает все поля.
Если ошибок нет — считает расстояние, цену и возвращает заказ со статусом «Создан».

### Factory Method

```go
type DeliveryMethod interface {        // Product
    Title() string
    BaseFee() Money
    MaxWeightKg() float64
    NextStatuses(from OrderStatus) []OrderStatus
    ...
}

type DeliveryCreator interface {       // Creator
    CreateMethod() DeliveryMethod      // фабричный метод
}

type LockerCreator struct{}            // ConcreteCreator
func (LockerCreator) CreateMethod() DeliveryMethod { return lockerDelivery{} }
```

`NewDeliveryMethod(t)` берёт создателя из карты `creators` по типу доставки и вызывает его.
Дальше код работает только с интерфейсом `DeliveryMethod`. Например, заказ спрашивает
у своего способа доставки, какие статусы можно ставить дальше:

```go
func (o *Order) NextStatuses() []OrderStatus {
    method, _ := NewDeliveryMethod(o.DeliveryType)
    return method.NextStatuses(o.Status)
}
```

### Strategy

```go
type PricingStrategy interface {
    Code() string
    Title() string
    Calculate(weightKg, distanceKm float64) Money
}
```

`WeightPricing` и `DistancePricing` — две реализации. Список `PricingStrategies` —
все стратегии: по нему строится выбор в форме и ищется стратегия по коду.

Выбор курьера устроен так же: `CourierSelector` → `NearestCourier` и `LeastLoadedCourier`.
Внутри — простой цикл «найти лучшего свободного курьера».

### Decorator

```go
type PriceComponent interface {
    Total() Money
    Lines() []PriceLine
}

type fragile struct{ inner PriceComponent }      // обёртка хранит то, что оборачивает

func (e fragile) Total() Money { return e.inner.Total() + 400 }
```

`CalculatePrice` создаёт базовую цену и по очереди оборачивает её выбранными услугами:

```go
var price PriceComponent = basePrice{...}
for _, extra := range Extras {
    if slices.Contains(in.Extras, extra.Code) {
        price = wrap(price, extra.Code, in.DeclaredValue)
    }
}
```

Получается «матрёшка»: `express( fragile( insurance( base ) ) )`.
`Total()` внешней обёртки идёт внутрь до базы и на обратном пути добавляет стоимость каждой услуги.

### Ошибки

- `ValidationError` — ошибки полей формы → HTTP `422`.
- `NotFound(...)` → `404`.
- `RuleViolation(...)` — нарушено правило (например, недопустимый статус) → `409`.

---

## 4. Application — сценарии

### port — интерфейсы

```go
type Repository[T any] interface {
    List() ([]T, error)
    Get(id string) (T, error)
    Save(item T) error
    Delete(id string) error
}

type Notifier interface { Notify(n Notification) error }
```

Application говорит, **что** ему нужно, но не знает, **кто** это сделает.
Реализации лежат в `adapters/out`. Поэтому в тестах SMS можно заменить заглушкой.

### events — Observer

```go
type Observer interface {
    Update(e Event)
}

type Publisher struct {
    observers []Observer
}

func (p *Publisher) Subscribe(o Observer) { p.observers = append(p.observers, o) }

func (p *Publisher) Notify(e Event) {
    for _, o := range p.observers {
        o.Update(e)
    }
}
```

- `Publisher` — список наблюдателей. Команда вызывает `Notify`, каждый получает событие.
- Наблюдатели (`subscribers.go`):
  - `ActivityRecorder` — пишет событие в журнал;
  - `CustomerNotifier` — при смене статуса отправляет клиенту SMS.
- Подписка — в `main.go`.

### command — команды

`OrderCommands`: `Create`, `Update`, `Delete`, `AssignCourier`, `ChangeStatus`.
`CustomerCommands`, `CourierCommands`: добавить, изменить, удалить.

Каждая команда делает одно и то же:
1. достать объект из хранилища;
2. вызвать метод domain (он проверит правила);
3. сохранить;
4. оповестить наблюдателей.

```go
func (c *OrderCommands) AssignCourier(id, selectorCode string) error {
    order, _ := c.orders.Get(id)
    selector, _ := domain.CourierSelectorByCode(selectorCode) // Strategy
    couriers, _ := c.couriers.List()
    courier, _ := selector.Select(order, couriers)
    order.AssignCourier(courier, c.now())                     // правило в domain
    c.couriers.Save(courier)
    c.orders.Save(order)
    c.notify(events.OrderStatusChanged, id, "...")            // Observer
    return nil
}
```
(в настоящем коде после каждого шага проверяется `err`.)

### query — запросы

- `OrderQueries.List / Get` — заказ + имена клиента и курьера + какие кнопки доступны
  (`nextStatuses`, `editable`, `deletable`). Фронтенд просто показывает то, что пришло.
- `OrderQueries.Quote` — цена по всем стратегиям сразу, без сохранения (для формы).
- `OrderQueries.DailyReport` — отчёт за день; `report.CSV()` — выгрузка в Excel.
- `DirectoryQueries` — клиенты, курьеры, точки, журнал, справочники для форм.

### facade.go — Facade

```go
func (f *DeliveryFacade) ChangeOrderStatus(id, status string) (query.OrderView, error) {
    if err := f.orderCmd.ChangeStatus(id, status); err != nil {   // команда
        return query.OrderView{}, err
    }
    return f.orders.Get(id)                                       // запрос
}
```

HTTP вызывает один метод фасада и получает готовый ответ.
О командах, запросах и хранилище HTTP не знает.

---

## 5. Adapters — края приложения

### in/http — входной адаптер

- `router.go` — список маршрутов: `"POST /api/orders"` → обработчик.
- `handlers.go` — обработчики. Каждый: прочитать запрос → вызвать фасад → вернуть результат.
  ```go
  func (a *api) createOrder(r *http.Request) (any, error) {
      var in command.OrderInput
      if err := readJSON(r, &in); err != nil {
          return nil, err
      }
      return a.facade.CreateOrder(in)
  }
  ```
- `respond.go` — `handler` превращает `(данные, ошибка)` в JSON-ответ;
  `writeError` выбирает код по виду ошибки (422 / 404 / 409 / 400 / 500).

### out/storage — хранилище

- `Collection[T]` — один JSON-файл на тип: в памяти `map[id]T`.
  При запуске файл читается, после `Save` / `Delete` перезаписывается целиком.
- `Get` и `List` отдают **копии** — объект меняется в хранилище только через `Save`.
- `sync.Mutex` — сервер обрабатывает запросы параллельно, мьютекс не даёт им
  одновременно менять `map`.
- `ActivityLog` — журнал в `activity.json`, `ExportFolder` — отчёты в `exports/`.

### out/notify — Adapter

```go
// Adaptee: «чужой» SMS-сервис — номер без «+», ответ — код
func (g *SmsGateway) Send(number, text string) int

// Adapter: реализует наш port.Notifier
func (a *SmsNotifier) Notify(n port.Notification) error {
    number := strings.TrimPrefix(n.Phone, "+")   // +7701… → 7701…
    switch a.gateway.Send(number, text) {        // код → error
    case 200:
        return nil
    case 400:
        return fmt.Errorf("шлюз не принял номер %s", n.Phone)
    default:
        return fmt.Errorf("SMS-шлюз недоступен")
    }
}
```

«Отправленные» SMS пишутся в `backend/data/sms_outbox.txt`.

---

## 6. main.go — сборка

```go
store := storage.Open(dataDir)                          // 1. хранилище
publisher := events.NewPublisher()                      // 2. Observer
publisher.Subscribe(events.NewActivityRecorder(...))
publisher.Subscribe(events.NewCustomerNotifier(..., sms, ...))
facade := application.NewDeliveryFacade(Deps{...})      // 3. фасад
if store.IsEmpty() { seed(facade, store) }              // 4. тестовые данные
http.ListenAndServe(addr, httpadapter.NewRouter(...))   // 5. HTTP
```

Единственное место, где встречаются все слои. Остальной код получает зависимости
через параметры — поэтому в тестах их легко заменить.

---

## 7. Фронтенд — `frontend/src`

| Файл | Что делает |
|---|---|
| `lib/api.ts` | типы данных и все запросы к серверу |
| `lib/app.tsx` | справочники для форм и `refresh()` — после действия все таблицы перезагружаются |
| `pages/orders/*` | таблица заказов, форма заказа (с живым расчётом цены), карточка заказа |
| `pages/CustomersPage.tsx`, `CouriersPage.tsx` | клиенты и курьеры |
| `pages/ActivityPage.tsx`, `ReportPage.tsx` | журнал и отчёт |

Бизнес-правил на фронтенде нет: какие кнопки показать и сколько стоит заказ, решает сервер.

---

## 8. Тесты

```bash
cd backend && go test ./... -v
```

| Файл | Что проверяет |
|---|---|
| `domain/domain_test.go` | Builder, обе Strategy, Decorator, правила статусов (Factory Method) |
| `application/facade_test.go` | полный сценарий + Observer, сохранение и перезапуск, бизнес-правила |
| `adapters/out/notify/sms_test.go` | Adapter |

---

## 9. Что могут попросить изменить на защите

| Попросят | Что сделать |
|---|---|
| Новая стратегия цены | `domain/pricing.go`: тип с `Code`, `Title`, `Calculate` → добавить в `PricingStrategies` |
| Новая стратегия курьера | `domain/courier_selection.go`: тип с `Select` → добавить в `CourierSelectors` |
| Новая доп. услуга (Decorator) | `domain/price.go`: тип-обёртка с `Total` и `Lines` → `case` в `wrap` → запись в `Extras` |
| Новый способ доставки | `domain/delivery.go`: новый продукт + новый `Creator` → в `creators` и `DeliveryTypes` |
| Новый наблюдатель | тип с методом `Update(e events.Event)` → `publisher.Subscribe(...)` в `main.go` |
| Email вместо SMS | `adapters/out/notify`: тип с `Notify(port.Notification) error` → передать в `main.go` |
| Новый лимит веса | `MaxWeightKg()` нужного способа в `delivery.go` |

После правки: `go test ./...` и перезапуск сервера.
